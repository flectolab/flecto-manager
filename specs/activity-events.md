# Spec — Journal d'activité par projet

Statut : à valider · Cible : `activity_events` + affichage par projet

## 1. Objectif

Tracer, par projet, qui a fait quoi sur les redirections et les pages, et l'afficher
dans le webui. La table ne doit pas croître indéfiniment : elle est bornée par un
nombre maximum d'events par projet, ce qui donne une taille maximale garantie de
`nb_projets × max_events_per_project` lignes.

Hors périmètre : namespaces, projets, utilisateurs, rôles, tokens, agents. Le
journal n'est pas un système de versioning — il ne permet pas de restaurer un état
(voir §12).

## 2. Décisions actées

| Sujet | Décision |
|---|---|
| Granularité | 1 event par action utilisateur ; les opérations de masse (import, publish, rollback projet) produisent **1 seul** event agrégé |
| Events répétés | Aucune coalescence : 5 modifications = 5 events |
| Périmètre | `REDIRECT`, `PAGE`, `PROJECT` (publish) uniquement |
| Rollback | Une seule action `ROLLBACK`, le payload distingue `scope: PROJECT` (global) de `scope: SINGLE` (unitaire) |
| Suppression de projet | Cascade DB (FK `ON DELETE CASCADE`), voir §3.3 |
| Droits de lecture | Accès en lecture au projet, sans nouvelle permission (voir §8) |
| Purge | Plafond du nombre d'events par projet, **sans** critère d'âge (§9), exécutée par un scheduler générique — **implémentée en dernier** |

## 3. Modèle de données

### 3.1 Contexte technique vérifié

Le schéma réel est produit par les migrations Atlas/golang-migrate
(`migrations/*.sql`, appliquées par `flecto-manager db migrate apply`), **pas** par
`AutoMigrate`. `database.Models` sert de source au générateur
(`tools/atlas-loader/main.go`) et aux tests (`AutoMigrate` sur SQLite in-memory).

`gormschema` génère les FK projet **dans le mauvais sens** (c'est `projects` qui
référence `redirects`) et sans `ON DELETE CASCADE`. Le loader Atlas corrige ça :
les mauvaises contraintes sont listées dans `removeConstraints` et remplacées par
celles de `customForeignKeys`, qui portent le `ON DELETE CASCADE`. Le cascade sur
`fk_redirects_project` est donc bien intentionnel et présent en base — aucun tag
n'a été perdu dans le code.

SQLite est historique : le dialecteur n'est plus enregistré dans `FactoryDialector`
(pas d'`init()` dans `database/sqlite.go`, contrairement à `database/mysql.go`), donc
il ne sert plus qu'aux tests via `gorm.Open(sqlite.Open(":memory:"))`. Conséquence
pour cette spec : les types de colonnes visent MariaDB/MySQL, mais doivent rester
acceptés par `AutoMigrate` SQLite pour les tests (`longtext` convient aux deux).

### 3.2 Table `activity_events`

```go
// model/activity_event.go
type ActivityResource string
type ActivityAction string

const (
    ActivityResourceRedirect ActivityResource = "REDIRECT"
    ActivityResourcePage     ActivityResource = "PAGE"
    ActivityResourceProject  ActivityResource = "PROJECT"

    ActivityActionCreate   ActivityAction = "CREATE"
    ActivityActionUpdate   ActivityAction = "UPDATE"
    ActivityActionDelete   ActivityAction = "DELETE"
    ActivityActionImport   ActivityAction = "IMPORT"
    ActivityActionRollback ActivityAction = "ROLLBACK"
    ActivityActionPublish  ActivityAction = "PUBLISH"
)

var ActivityEventSortableColumns = map[string]string{
    "occurredAt": "occurred_at",
    "resource":   "resource",
    "action":     "action",
    "actor":      "actor",
}

type ActivityEvent struct {
    ID            int64  `json:"id" gorm:"primaryKey;autoIncrement"`
    NamespaceCode string `json:"-" gorm:"size:50;index:idx_activity_events_ns_proj,priority:1"`
    ProjectCode   string `json:"-" gorm:"size:50;index:idx_activity_events_ns_proj,priority:2"`

    Resource ActivityResource `json:"resource" gorm:"size:20;not null"`
    Action   ActivityAction   `json:"action" gorm:"size:20;not null"`

    // Actor is a snapshot: username for a JWT session, token name for an API token.
    // Deliberately not a foreign key, so the trail survives user deletion or rename.
    UserID   *int64         `json:"userID" gorm:"index:idx_activity_events_user"`
    Actor    string         `json:"actor" gorm:"size:300;not null"`
    AuthType types.AuthType `json:"authType" gorm:"size:20"`

    // ResourceID is the redirect/page id when the event targets a single entry.
    ResourceID *int64    `json:"resourceID" gorm:"index:idx_activity_events_resource"`
    Data       ActivityData `json:"data" gorm:"type:longtext"`

    OccurredAt time.Time `json:"occurredAt" gorm:"type:timestamp;not null"`
}

type ActivityEventList = commonTypes.PaginatedResult[ActivityEvent]
```

Justifications :

- **`Resource` + `Action` séparés** plutôt qu'un `type` unique : filtrage natif
  (« tous les publish », « tout ce qui touche aux pages ») sans `LIKE`, et
  cohérence avec `model.ResourceType`/`ActionType` de `permission.go`. Côté front,
  la clé d'affichage reste unique : `` `${resource}_${action}` ``.
- **Acteur dénormalisé** : `UserID` sert au filtrage et aux liens, `Actor` garantit
  la lisibilité du journal même après suppression de l'utilisateur.
  `auth.GetUser(ctx)` renvoie `UserID: 0` pour un token API → à mapper vers `nil`.
- **Suppression d'un utilisateur** : `user_id` passe à `NULL`, `actor` est conservé.
  Le journal continue d'afficher « alice », il perd seulement le lien vers la fiche
  utilisateur. C'est la raison d'être du snapshot `Actor` — voir §3.3 pour le
  mécanisme (FK `ON DELETE SET NULL`).
- **Pas de champ `Project *Project` ni `User *User`** : sans association GORM,
  `gormschema` ne génère aucune FK à retirer, et les deux vraies contraintes sont
  ajoutées à la main (§3.3).
- **Tri sur `id`** : deux events écrits dans la même transaction partagent la même
  `occurred_at`. `id DESC` est l'ordre total stable, et le curseur de purge.
- **`OccurredAt` et pas `CreatedAt`/`UpdatedAt`** : la ligne est immuable, pas de
  soft-delete, pas de mise à jour.
- **Index** : `idx_activity_events_ns_proj` couvre à la fois la liste paginée, le
  filtre par période et la purge. Deux colonnes suffisent : en InnoDB la clé
  primaire est implicitement ajoutée aux feuilles de tout index secondaire, donc
  l'`ORDER BY id DESC` qui suit l'égalité sur (ns, proj) est déjà ordonné par
  l'index, sans filesort. Même forme que `idx_redirects_namespace_project`.
  Pas d'index sur `occurred_at` seul : il n'aurait servi qu'à une purge par âge,
  écartée (§9), et le filtre `from`/`to` s'applique toujours à une partition d'au
  plus `max_events_per_project` lignes.

### 3.3 Migration

1. Ajouter `model.ActivityEvent{}` à `database.Models` (`database/db.go`).
2. Ajouter dans `tools/atlas-loader/main.go`, à `customForeignKeys` :
   ```go
   "ALTER TABLE `activity_events` ADD CONSTRAINT `fk_activity_events_project` FOREIGN KEY (`namespace_code`,`project_code`) REFERENCES `projects`(`namespace_code`,`project_code`) ON DELETE CASCADE;",
   "ALTER TABLE `activity_events` ADD CONSTRAINT `fk_activity_events_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE SET NULL;",
   ```
   Rien à ajouter à `removeConstraints` puisque le modèle ne déclare aucune association.
3. Générer la migration : `./bin/atlas-diff.sh add_activity_events`, puis
   `atlas migrate hash --env local --config file://./tools/atlas-loader/atlas.hcl`.
4. Vérifier dans le SQL généré : `namespace_code`/`project_code` en `varchar(50)`
   **NOT NULL** (obligatoire pour une FK composite fiable), `user_id` en `bigint`
   **NULL**, `data` en `longtext`, les deux index, et les deux contraintes avec le
   bon `ON DELETE`.

**Suppression de projet → `CASCADE`** : le journal du projet part avec lui. C'est
voulu, sinon un projet recréé avec le même code hériterait de l'historique de
l'ancien.

**Suppression d'utilisateur → `SET NULL`** : la ligne d'activité survit, `user_id`
devient `NULL`, `actor` garde le nom. Le faire au niveau de la base plutôt qu'en
Go est délibéré : `userService.Delete` (`service/user_service.go:125`) enchaîne
suppression du rôle puis de l'utilisateur **sans transaction**, donc un `UPDATE`
applicatif ajouterait une troisième étape non atomique, et n'importe quel futur
chemin de suppression pourrait l'oublier. Une FK à `users` est par ailleurs déjà le
pattern en place (`fk_user_roles_user`, en `CASCADE`).

Limite à connaître : cette FK n'existe pas sous SQLite (aucune association GORM
donc aucune FK créée par `AutoMigrate`, et SQLite n'applique pas les FK sans
`PRAGMA foreign_keys=ON`). Le comportement n'est donc pas vérifiable par les tests
unitaires in-memory — à basculer en code applicatif si on veut un jour un test
automatisé.

Migration générée : `migrations/20260818064321_add_activity_events.{up,down}.sql`.
Les deux comportements ont été vérifiés sur MariaDB dans un schéma jetable :
suppression de l'utilisateur → `user_id` à `NULL` et `actor` conservé ;
suppression du projet → ligne d'activité supprimée.

### 3.4 Type `ActivityData`

```go
// ActivityData is a raw JSON payload whose shape is determined by (Resource, Action).
// It is written by the server only: UnmarshalGQL always fails.
type ActivityData json.RawMessage

func (d ActivityData) Value() (driver.Value, error)      // string(d), nil si vide
func (d *ActivityData) Scan(value any) error             // []byte / string
func (d ActivityData) MarshalGQL(w io.Writer)            // passthrough, "null" si vide
func (d *ActivityData) UnmarshalGQL(v any) error         // toujours une erreur
```

`Value`/`Scan` explicites évitent toute ambiguïté GORM sur un type nommé dérivé de
`[]byte`. `MarshalGQL` écrit le JSON tel quel dans la réponse — sûr, puisque le
contenu vient de `json.Marshal` côté serveur.

## 4. Payloads

Dans `model/activity_payload.go`. Règle absolue : **le contenu d'une page n'entre
jamais dans un payload** (jusqu'à 1 Mo par page via `page.size_limit`).

```go
// RedirectSnapshot is the activity-trail projection of a redirect.
type RedirectSnapshot struct {
    Type   commonTypes.RedirectType   `json:"type"`
    Source string                     `json:"source"`
    Target string                     `json:"target"`
    Status commonTypes.RedirectStatus `json:"status"`
}

// PageSnapshot is the activity-trail projection of a page. Content is never included.
type PageSnapshot struct {
    Type        commonTypes.PageType        `json:"type"`
    Path        string                      `json:"path"`
    ContentType commonTypes.PageContentType `json:"contentType"`
    ContentSize int64                       `json:"contentSize"`
}

// ActivityChange carries the before/after of a single-entry change.
// Before is nil on a creation, After is nil on a deletion.
type ActivityChange[T any] struct {
    Before *T `json:"before,omitempty"`
    After  *T `json:"after,omitempty"`
}

type ActivityRollbackScope string

const (
    ActivityRollbackScopeSingle  ActivityRollbackScope = "SINGLE"
    ActivityRollbackScopeProject ActivityRollbackScope = "PROJECT"
)

// ActivityRollback describes a discarded pending change. SINGLE fills ChangeType and
// Entry, PROJECT fills Discarded.
type ActivityRollback[T any] struct {
    Scope      ActivityRollbackScope `json:"scope"`
    ChangeType *DraftChangeType   `json:"changeType,omitempty"`
    Entry      *T                 `json:"entry,omitempty"`
    Discarded  *ActivityDraftCounts  `json:"discarded,omitempty"`
}

type ActivityDraftCounts struct {
    Create int64 `json:"create"`
    Update int64 `json:"update"`
    Delete int64 `json:"delete"`
}

// ActivityImportErrorSampleMax bounds the errors kept in the payload. ErrorCount stays
// exact, so a truncated sample is detectable with errorCount > len(errorSample).
const ActivityImportErrorSampleMax = 20

type ActivityImport struct {
    Filename    string              `json:"filename"`
    Overwrite   bool                `json:"overwrite"`
    TotalLines  int                 `json:"totalLines"`
    Imported    int                 `json:"imported"`
    Skipped     int                 `json:"skipped"`
    ErrorCount  int                 `json:"errorCount"`
    ErrorSample []ActivityImportError  `json:"errorSample,omitempty"`
}

type ActivityImportError struct {
    Line   int    `json:"line"`
    Source string `json:"source,omitempty"`
    Reason string `json:"reason"`
}

type ActivityPublish struct {
    Version   int                `json:"version"`
    Redirects ActivityPublishCounts `json:"redirects"`
    Pages     ActivityPublishCounts `json:"pages"`
}

type ActivityPublishCounts struct {
    Created int `json:"created"`
    Updated int `json:"updated"`
    Deleted int `json:"deleted"`
}
```

Les génériques sont cohérents avec l'existant (`commonTypes.PaginatedResult[T]`).

## 5. Catalogue des events

`RedirectDraftService.Create` produit **trois** actions différentes selon ses
arguments (`service/redirect_draft_service.go:60`) : c'est le point d'émission le
plus subtil du lot. L'action reflète **l'intention de l'utilisateur**, pas le
`ChangeType` du draft.

| Appel | Event | Payload |
|---|---|---|
| `Create(oldID=nil, new=X)` | `REDIRECT/CREATE` | `ActivityChange{After: X}` |
| `Create(oldID=42, new=X)` | `REDIRECT/UPDATE` | `ActivityChange{Before: live(42), After: X}` |
| `Create(oldID=42, new=nil)` | `REDIRECT/DELETE` | `ActivityChange{Before: live(42)}` |
| `Update(draftID, new=X)` | `REDIRECT/UPDATE` | `ActivityChange{Before: draft.NewRedirect, After: X}` |
| `Delete(draftID)` | `REDIRECT/ROLLBACK` | `ActivityRollback{Scope: SINGLE, ChangeType: draft.ChangeType, Entry: …}` |
| `Rollback(ns, proj)` | `REDIRECT/ROLLBACK` | `ActivityRollback{Scope: PROJECT, Discarded: {c,u,d}}` |
| `RedirectImportService.Import` | `REDIRECT/IMPORT` | `ActivityImport` |
| idem pour les pages (`PageDraftService`) | `PAGE/*` | `PageSnapshot` au lieu de `RedirectSnapshot`, pas d'`IMPORT` |
| `ProjectService.Publish` | `PROJECT/PUBLISH` | `ActivityPublish` |

`ResourceID` = l'id de la redirect/page concernée (`draft.OldRedirectID` /
`draft.OldPageID`) pour les events unitaires ; `nil` pour `IMPORT`, `ROLLBACK
PROJECT` et `PUBLISH`.

Lecture du journal attendue :

```
09:12  alice   REDIRECT/CREATE     /old → /new (301)
09:14  alice   REDIRECT/UPDATE     /a → /b   ⇒   /a → /c
09:15  bob     REDIRECT/ROLLBACK   annulé : création de /old
09:20  alice   PROJECT/PUBLISH     v12 · redirects +1 ~1
```

## 6. Service et points d'émission

### 6.1 `service/activity_service.go`

```go
// model/activity_payload.go
type ActivityInput struct {
    NamespaceCode string
    ProjectCode   string
    Resource      ActivityResource
    Action        ActivityAction
    ResourceID    *int64
    Data          any // marshalled when recorded
}

// service/activity_service.go
type ActivityService interface {
    GetTx(ctx context.Context) *gorm.DB
    GetQuery(ctx context.Context) *gorm.DB
    // Record persists an event. tx must be the transaction of the recorded
    // operation so the event cannot survive a rollback; pass nil to write outside
    // any transaction.
    Record(ctx context.Context, tx *gorm.DB, in model.ActivityInput) error
    Search(ctx context.Context, query *gorm.DB) ([]model.ActivityEvent, error)
    SearchPaginate(ctx context.Context, pagination *commonTypes.PaginationInput, query *gorm.DB) (*model.ActivityEventList, error)
    Purge(ctx context.Context) (int64, error) // §9, ajouté au dernier lot
}
```

`Record` :
1. résout l'acteur via `usercontext.GetUser(ctx)` ; `UserID == 0` → `nil` ; si le
   contexte n'a pas d'utilisateur (chemin interne, CLI), écrire
   `Actor: model.ActivityActorSystem` plutôt que d'échouer ;
2. `json.Marshal(in.Data)` ;
3. `OccurredAt = time.Now()` ;
4. insert via `tx` s'il est non nil, sinon via le repository.

**Deux cycles d'import à contourner**, découverts à l'implémentation :

- `auth` importe `service` (middleware et permission checker prennent des services),
  donc `service` ne peut pas importer `auth` pour lire l'acteur. `UserContext`,
  `GetUser` et `SetUserContext` sont extraits dans un package feuille
  `auth/usercontext`, qui n'importe que `model` et `types`. Le package `auth` garde
  un alias de type et deux fonctions de délégation, donc aucun appelant existant
  (resolvers, routes REST) ne change.
- `ActivityInput` vit dans `model`, pas dans `service` : les mocks de services sont
  générés dans un package que les tests de `service` importent, donc une interface
  de service exposant un type `service.X` ferait cycler ces tests. C'est déjà la
  raison pour laquelle `RedirectImportService` n'est pas mocké (`ImportRedirectOptions`).

**Politique d'échec** : l'écriture se fait dans la transaction de l'opération, donc
un échec d'enregistrement fait échouer l'opération tracée. C'est le bon défaut pour un
journal d'activité — pas de trou silencieux. À rediscuter seulement si un incident le
justifie.

### 6.2 Fichiers à modifier

| Fichier | Modification |
|---|---|
| `service/redirect_draft_service.go` | `Create`/`Update`/`Delete`/`Rollback` : `Record` dans la transaction existante |
| `service/page_draft_service.go` | idem |
| `service/redirect_import_service.go` | `Record` dans la transaction de `Import` |
| `service/project_service.go` | `Record` dans la transaction de `Publish` |
| `service/services.go` | nouveau champ `Activity`, injection dans les 4 services ci-dessus |

Conséquence mécanique : les signatures `NewRedirectDraftService`,
`NewPageDraftService`, `NewRedirectImportService`, `NewProjectService` changent →
tous les tests existants de ces services doivent être ajustés. Ils reçoivent un
**vrai** service d'activité branché sur leur base de test (`newTestActivityService`)
plutôt qu'un mock : les événements écrits sont alors assertables directement, et
aucune attente `EXPECT().Record(...)` n'a à être ajoutée à chaque test existant.

Effet de bord assumé sur `Update` : la sauvegarde du draft passait par
`repo.Update`, qui ouvre sa propre connexion et ne peut pas rejoindre une
transaction externe. Elle se fait désormais par `tx.Save` dans la transaction, pour
que le draft et son événement d'activité soient validés ensemble. C'est déjà ce que
font `Create`, `Delete` et `Rollback`. Les trois sous-tests qui simulaient l'échec
via `EXPECT().Update(...).Return(err)` le font maintenant par un callback GORM sur
la table des drafts, et vérifient en plus qu'aucun événement ne survit à l'échec.

### 6.3 Détails d'implémentation par point d'émission

- **`RedirectDraftService.Create` en `UPDATE`/`DELETE`** : le `before` n'est pas
  chargé aujourd'hui (seul `oldRedirectID` est connu). Charger la redirect dans la
  transaction (`tx.First(&model.Redirect{}, *oldRedirectID)`). Si elle n'est pas
  encore publiée (`Redirect.Redirect == nil`), `Before` reste `nil`. Coût : +1
  SELECT par mutation de draft, négligeable au rythme humain.
- **`PageDraftService`** : charger le `before` **sans le contenu** —
  `tx.Select("id","type","path","content_type","content_size")` — pour ne pas
  tirer 1 Mo en mémoire à chaque event. Le journal dit que le contenu a changé et
  sa nouvelle taille, pas le diff.
- **`RedirectDraftService.Rollback`** (`redirect_draft_service.go:178`) : ne compte
  rien avant de supprimer. Ajouter un `SELECT change_type, COUNT(*) … GROUP BY
  change_type` dans la transaction, avant les `DELETE`.
- **`RedirectImportService.Import`** : le nom du fichier n'est pas connu du service.
  Ajouter `Filename string` à `ImportRedirectOptions` et le renseigner dans le
  resolver depuis `graphql.Upload.Filename`. L'échantillon d'erreurs se dérive de
  `result.Errors` en le tronquant à `ActivityImportErrorSampleMax`.
- **`ProjectService.Publish`** : les compteurs se déduisent des boucles existantes
  sur `redirectDrafts`/`pageDrafts` (`project_service.go:209` et suivantes) en
  comptant par `ChangeType`. `Version` = `project.Version` **après** incrément.

## 7. API GraphQL

`graph/schema/activity.graphqls` :

```graphql
scalar JSON

enum ActivityResource { REDIRECT PAGE PROJECT }
enum ActivityAction { CREATE UPDATE DELETE IMPORT ROLLBACK PUBLISH }

type ActivityEvent {
    id: Int64!
    resource: ActivityResource!
    action: ActivityAction!
    userID: Int64
    actor: String!
    authType: String!
    resourceID: Int64
    data: JSON
    occurredAt: DateTime!
}

type ActivityEventList {
    items: [ActivityEvent!]!
    total: Int!
    limit: Int!
    offset: Int!
}

input ActivityEventFilter {
    resource: ActivityResource
    action: ActivityAction
    actor: String
    from: DateTime
    to: DateTime
}

extend type Query {
    projectActivityEvents(
        namespaceCode: String!
        projectCode: String!
        pagination: PaginationInput
        filter: ActivityEventFilter
    ): ActivityEventList!
}
```

`gqlgen.yml` :

```yaml
  JSON:
    model: github.com/flectolab/flecto-manager/model.ActivityData
  ActivityEvent:
    model: github.com/flectolab/flecto-manager/model.ActivityEvent
  ActivityEventList:
    model: github.com/flectolab/flecto-manager/model.ActivityEventList
  ActivityResource:
    model: github.com/flectolab/flecto-manager/model.ActivityResource
  ActivityAction:
    model: github.com/flectolab/flecto-manager/model.ActivityAction
```

Puis `go tool gqlgen generate`.

`authType` reste un `String!` et non une enum : gqlgen ne sait pas lier
`types.AuthType` au scalaire `String`, il génère donc un field resolver de deux
lignes. C'est voulu plutôt que corrigé par une enum, car la colonne est vide pour
les événements enregistrés hors d'une requête authentifiée (CLI, `db demo`), et
aucune valeur d'enum ne peut porter cette absence.

`limit/offset` suffit (pas de keyset) : la table est bornée par projet par la purge.
Tri fixe `id DESC` ; l'ordre est un détail d'implémentation du journal, pas une
option utilisateur.

## 8. Permissions

Dans le resolver, avant toute requête :

```go
if !r.PermissionChecker.CanResource(userCtx.SubjectPermissions, namespaceCode, projectCode,
    model.ResourceTypeAny, model.ActionRead) {
    return nil, fmt.Errorf("user %s has no permission to access project %s/%s", userCtx.Username, namespaceCode, projectCode)
}
```

`ResourceTypeAny` est traité comme un joker côté ressource par
`matchResource` (`auth/permission_checker.go:118`) : « une permission de lecture sur
n'importe quelle ressource du projet suffit ». C'est déjà le pattern de la query
`project` (`graph/resolver/project.resolvers.go:129`).

**Aucune nouvelle `ResourceType`, aucun changement dans l'éditeur de rôles.**

Contrepartie assumée : un utilisateur qui n'a que `redirect:read` verra les events
`PAGE/*` (chemin et taille, jamais le contenu). Si ça devient un problème, filtrer
les lignes par `resource` selon les permissions — pas fait maintenant.

## 9. Purge et scheduler (dernier lot)

### 9.1 Scheduler générique

Nouveau package `scheduler/`, conçu pour accueillir d'autres tâches de fond :

```go
// Task is a unit of recurring background work.
type Task interface {
    Name() string
    Interval() time.Duration   // 0 disables the task
    RunOnStart() bool          // run once at boot instead of waiting for the first tick
    Timeout() time.Duration    // 0 leaves a run unbounded
    Run(ctx context.Context) error
}

type Scheduler struct { /* appCtx, tasks, cancel, wg */ }

func New(appCtx *appContext.Context) *Scheduler
func (s *Scheduler) Register(tasks ...Task)
func (s *Scheduler) Start()
func (s *Scheduler) Stop()   // cancel, then wait for the runs in flight
func (s *Scheduler) Wait()
```

**Une seule interface, pas d'interfaces optionnelles.** `RunOnStart` et `Timeout`
avaient d'abord été sortis dans des interfaces facultatives, testées par assertion
de type dans le runner. Replié, pour deux raisons : la valeur zéro de chaque réglage
porte déjà la sémantique « pas d'option » (intervalle nul = tâche désactivée, timeout
nul = run non borné, `RunOnStart` faux = attendre le premier tick), et le découpage
ne coûtait que des assertions de type dans le runner et de la composition pénible
côté appelant — au point qu'enregistrer une tâche dans les tests demandait de
composer une struct anonyme. Le pattern d'interface optionnelle vaut quand
l'interface de base est figée dans un paquet tiers ; ici les deux côtés sont à nous.

Une goroutine + un `time.Ticker` par tâche. Deux garanties en découlent, mesurées et
verrouillées par des tests plutôt que déduites de la documentation :

- **Une tâche lente n'en retarde aucune autre** : les goroutines sont indépendantes.
- **Deux exécutions d'une même tâche ne se chevauchent jamais**, et les ticks qui
  tombent pendant une exécution sont **abandonnés**, pas empilés. Mesuré : avec un
  intervalle de 10 ms et des exécutions de 60 ms sur une fenêtre de 300 ms, on obtient
  5 exécutions et une concurrence maximale de 1, et non 30 exécutions en file. Une
  tâche peut donc toucher son propre état sans verrou.

Corollaire : le timeout n'a **pas** besoin d'être inférieur à l'intervalle pour la
correction, il n'y a ni course ni accumulation. Mais si le timeout est supérieur ou
égal à l'intervalle, une tâche lente peut enchaîner les exécutions sans jamais de
pause : le scheduler le signale par un avertissement au démarrage plutôt que de le
corriger en silence. Chaque exécution est
enveloppée d'un `recover()` (un panic ne doit pas tuer le process — ce que ne fait
pas `metrics.StartCollector` aujourd'hui), d'un `context.WithTimeout` si la tâche
implémente `TimeoutTask`, et d'un log uniforme `task` / `duration` / `error`.

`appContext` n'expose qu'un `done chan bool` (`context/context.go:36`) : le
scheduler dérive lui-même un `context.Context` annulé à la fermeture de ce canal.
Pas de modification du contexte applicatif.

Câblage dans `cli/start.go`, à côté du serveur metrics :

```go
sched := scheduler.New(ctx)
sched.Register(task.NewActivityPurge(ctx, services.Activity))
sched.Start()
```

**File d'attente de jobs, à venir.** Le besoin annoncé — purger le cache CDN des
redirects et pages déplacés par un publish — n'est pas une tâche récurrente mais un
**job produit par une opération et consommé en arrière-plan**. Cela ne remet pas en
cause l'abstraction : un consommateur de file est lui-même une `Task`, avec un
intervalle court, qui draine une table de jobs. Ce qu'il faudra ajouter, c'est la
table et l'enfilement, pas un second mécanisme.

Trois points à trancher à ce moment-là :

- **Durabilité** : le job doit être écrit dans la transaction du publish, exactement
  comme l'événement d'activité. Si le publish échoue, pas de purge de cache ; s'il
  réussit, le job existe forcément. Le levier est déjà en place (§6.1).
- **Taille du payload** : un publish peut déplacer 50 000 redirects. Soit le job
  porte la liste des URL — et il faut la borner — soit il porte le projet et sa
  version et le consommateur fait une purge par tag ou par joker, ce que la plupart
  des CDN savent faire. C'est la vraie question de conception, elle décide de tout
  le reste.
- **Reprise** : compteur de tentatives, `next_attempt_at`, backoff, état terminal.
  Une API de CDN échoue de façon transitoire, et la purge de cache est idempotente,
  donc l'at-least-once est le bon modèle.

Notes pour plus tard, hors lot :
`metrics.StartCollector` (`metrics/metrics.go:137`)
est une `Task` déguisée et serait le premier candidat à la migration ; et en
multi-réplicas chaque instance exécutera chaque tâche (inoffensif pour une purge
idempotente, à verrouiller — `GET_LOCK` ou table de lock — pour une future tâche
qui ne doit tourner qu'une fois).

### 9.2 `scheduler/task/activity_purge.go`

**Un seul critère : le plafond du nombre d'events par projet.**

Algorithme, par tick :

1. `SELECT DISTINCT namespace_code, project_code FROM activity_events` — ne touche que
   les projets qui ont effectivement des events, et l'index composite le sert.
2. Pour chacun, `SELECT id … WHERE ns=? AND proj=? ORDER BY id DESC LIMIT 1
   OFFSET N-1` donne le curseur ; aucune ligne renvoyée = le projet est sous le
   plafond, rien à faire.
3. `DELETE … WHERE ns=? AND proj=? AND id < curseur`, par lots.

Deux requêtes par projet au lieu d'un `DELETE` avec sous-requête `LIMIT`, interdit
par MySQL.

**Le lotissement sélectionne les ids avant de supprimer**, il n'utilise pas
`DELETE … LIMIT`. Vérifié à l'implémentation : MariaDB honore ce `LIMIT` mais SQLite
**l'ignore silencieusement**, sans erreur — un `DELETE … LIMIT 2` sur 5 lignes en
supprime 5. Le lotissement aurait donc été inopérant sous les tests tout en
fonctionnant en production, exactement le genre d'écart qu'un test ne rattrape pas.

**Pourquoi pas de purge par âge.** Le plafond est le seul des deux critères qui
borne réellement la table : `nb_projets × max_events_per_project`. Une rétention par
âge ne garantit aucune taille maximale (un projet actif peut écrire 100 000 events
en 90 jours) et introduit une perte de données sans contrepartie — un projet peu
actif verrait son historique effacé alors que le plafond n'aurait jamais été
atteint. Le seul objectif que l'âge servirait vraiment est la minimisation des
données personnelles (le journal contient des noms d'utilisateurs) ; si cette
exigence arrive, elle s'ajoutera comme une boucle indépendante — plus un index sur
`occurred_at`, à créer à ce moment-là.

Corollaire : le plafond étant le seul garde-fou, il doit être choisi en connaissance
de cause. Compter ~600 octets par ligne : 1000 events/projet ≈ 600 ko par projet,
soit ~60 Mo pour 100 projets. Il y a de la marge pour être généreux.

Ajouter aussi une commande manuelle `flecto-manager db activity-purge` dans
`cli/db.go` pour l'ops.

### 9.3 Configuration

```yaml
activity:
  max_events_per_project: 1000  # 0 = illimité
  purge_interval: 1h
```

`config.ActivityConfig` + valeurs dans `config.DefaultConfig()`. `purge_interval: 0`
désactive la tâche ; `max_events_per_project: 0` la neutralise aussi, mais c'est
alors un choix explicite d'accepter une table non bornée.

Pas de clé `enabled` : l'enregistrement est toujours actif. La taille étant déjà
bornée par le plafond, un interrupteur n'aurait aucune justification de capacité,
et un journal d'activité qu'on peut couper en silence perd l'essentiel de sa valeur —
le mode de défaillance serait une instance mal configurée qui n'enregistre rien
sans que personne ne s'en aperçoive avant d'avoir besoin du journal.

**Métriques Prometheus du scheduler.** Aucune métrique propre au journal d'activité,
mais les tâches de fond sont instrumentées, ce qui vaut pour toute tâche future :

| Métrique | Type | Labels |
|---|---|---|
| `flecto_scheduler_task_runs_total` | Counter | `task`, `status` |
| `flecto_scheduler_task_duration_seconds` | Histogram | `task` |
| `flecto_scheduler_task_last_success_timestamp_seconds` | Gauge | `task` |

Trois décisions derrière ces trois métriques :

- **Un panic compte comme une erreur.** Le `recover()` du runner alimente le
  compteur d'erreurs, sinon une tâche qui crashe à chaque exécution passerait pour
  saine.
- **Les deux séries `status` sont publiées avant la première exécution**
  (`InitSchedulerTask`). Une série absente fait qu'une règle d'alerte ne correspond
  à rien, ce qui est indistinguable d'une tâche en bonne santé.
- **`last_success_timestamp` couvre ce qu'un compteur d'erreurs ne peut pas voir** :
  une tâche qui ne tourne plus du tout n'émet aucune erreur, elle cesse simplement
  d'émettre. La règle est `time() - last_success > quelques intervalles`.

Piège à connaître pour les règles d'alerte : **la fenêtre doit dépasser l'intervalle
de la tâche.** Avec `purge_interval: 1h`, un `increase(...[5m])` lit 0 même quand la
tâche est cassée, puisqu'elle ne s'exécute qu'une fois par heure.

## 10. Webui

- `webui/src/graphql/activity.graphql` : la query + `npm run codegen`.
- `webui/src/types/activity.ts` : union discriminée TS **miroir** des structs Go de
  §4, indexée par `` `${resource}_${action}` ``.
- `webui/src/components/activity/ActivityEventTable.tsx` : les lignes du journal,
  partagées par la page et le widget du dashboard. **Ressource et action ont chacune
  leur colonne** : réunies dans une seule cellule, leurs badges se décalaient selon
  la longueur du nom de ressource et rien ne s'alignait verticalement. Colonnes :
  date / acteur / ressource / action / détail.
- `webui/src/pages/Activity.tsx` : la table plus les filtres (ressource, action,
  acteur, période) et la pagination.
- `webui/src/components/activity/RecentActivityEvents.tsx` : les 15 événements les plus
  récents sur le dashboard, sans filtre ni pagination, avec un lien vers le journal
  complet. Le composant porte sa propre vérification de permission et ne rend rien
  si l'utilisateur n'a pas accès — le dashboard s'affiche pour toute permission sur
  le projet, y compris en écriture seule, donc afficher une erreur de permission à
  côté des statistiques serait un mauvais comportement.
- `webui/src/components/activity/` : un composant de rendu par type d'event, résolus
  par un registry `resource_action → composant`, avec un **fallback générique
  obligatoire** — sans lui, un event écrit par une version N+1 du serveur casse
  l'affichage d'un webui N.
- Route `activity` dans `App.tsx` (à côté de `agents`) + entrée de navigation dans
  `components/layout/Sidebar.tsx`, sous `PermissionGate`.

**Une ligne par événement.** La colonne Detail tient toujours sur une seule ligne :
chaque champ s'écrit `libellé: valeur`, et un champ modifié `libellé: avant → après`
avec l'ancienne valeur barrée. Les compteurs d'un publish ou d'un rollback projet
s'écrivent en clair — `redirects: 39 created · 0 updated · 0 deleted` — en listant
toujours les trois, y compris à zéro : une forme prévisible se lit plus vite.

Si la ligne dépasse, elle est **tronquée** et un bouton **View changes** ouvre le
détail complet dans une modale reprenant le gabarit de
`components/redirects/DiffModal.tsx` (table Field / Before / After). Le bouton
apparaît aussi, même sans troncature, quand le payload porte quelque chose qui n'a
pas sa place sur une ligne : les erreurs d'import et le JSON brut d'un type inconnu.

**Une description, deux présentations.** `describeActivityEvent.ts` traduit un
événement en liste de descripteurs `{label, before?, after}`, et c'est la seule
source de vérité : la ligne et la modale en dérivent toutes les deux. Ajouter un
type d'événement se fait en le décrivant là, sans écrire de composant de rendu.
C'est ce qui a remplacé les six composants d'un rendu par type.

Deux conséquences de forme :

- **La couleur ne code jamais l'action** : le libellé est atténué, la valeur est en
  texte principal, l'ancienne valeur est barrée. Utiliser du gris pour les
  suppressions faisait lire le même champ différemment d'une ligne à l'autre, alors
  que la pastille ACTION porte déjà cette information.
- **Un `UPDATE` n'affiche que les champs qui ont changé**, plus la source (ou le
  path) comme identité même inchangée. Le payload contient l'avant et l'après, la
  comparaison se fait au rendu.

**La table est en `table-fixed`**, avec des largeurs sur les quatre premières
colonnes. Ce n'est pas cosmétique : en layout automatique la cellule de détail
s'élargit à son contenu, ce qui annule la troncature et fait déborder la ligne hors
du tableau.

**Détection de la troncature** : `hooks/useIsTruncated.ts` compare `scrollWidth` à
`clientWidth`, avec un `ResizeObserver` pour suivre les redimensionnements et une
re-mesure sur `document.fonts.ready` — `ResizeObserver` observe la boîte et non le
contenu, donc une police web appliquée après le montage changerait la largeur du
texte sans déclencher de callback.

Le champ `data` étant un scalaire `JSON`, la cohérence Go ↔ TS n'est pas garantie
par le compilateur. Mitigation : les structs de payload sont la référence, les
types TS sont écrits en miroir dans un seul fichier. Si la dérive devient pénible,
basculer `data` sur une union GraphQL typée (`ActivityEventData = RedirectChange | …`)
— plus sûr, mais churn de schéma à chaque nouvel event.

## 11. Tests

- `repository/activity_event_repository_test.go` : CRUD, filtres, pagination, tri
  (SQLite in-memory, comme `repositories_test.go:12`).
- `service/activity_service_test.go` : acteur JWT / token API (`UserID: 0` → `nil`) /
  contexte sans utilisateur, sérialisation du payload.
- Tests d'émission dans chaque service concerné : un event, et un seul, avec le bon
  couple `(Resource, Action)` et le bon payload pour chaque branche de
  `Create`/`Update`/`Delete`/`Rollback`/`Import`/`Publish`.
- Test d'atomicité : forcer une erreur après le `Record` dans la transaction et
  vérifier qu'aucune ligne d'activité ne subsiste.
- Test de bornage : un import de plus de `ActivityImportErrorSampleMax` erreurs produit
  un `errorCount` exact et un `errorSample` tronqué.
- Test de non-fuite : un event `PAGE/*` ne contient jamais la clé `content`.
- Purge : projet sous le plafond (aucune suppression), projet au-dessus (seules les
  plus anciennes partent, le compte retombe exactement au plafond), table vide,
  plusieurs projets dont un seul dépasse, `max_events_per_project: 0`.
- Suppression d'un utilisateur (`user_id` → `NULL`, `actor` conservé) : non
  couvrable en test unitaire, la contrainte est portée par MariaDB uniquement.
  Vérifié manuellement, voir §3.3.
- Rappel : il n'y a aucun test dans `graph/resolver/` aujourd'hui — ne pas en
  introduire dans ce lot, mettre la logique testable dans les services.

Régénérer les mocks avec `./bin/mock.sh` après ajout du repository et du service.

## 12. Ce que cette table n'est pas

Ce n'est pas un système de versioning : les payloads sont tronqués, les pages n'ont
pas leur contenu, et la purge efface l'historique ancien. « Restaurer la version 11
du projet » demanderait une table de snapshots indexée par `project.Version` —
autre sujet, autre spec.

Le chaînage par hash (chaque ligne hachant la précédente, journal infalsifiable)
n'est pertinent que sous contrainte de conformité, et la purge en coupe la racine :
une fois les plus anciennes lignes supprimées, la chaîne n'est plus vérifiable
jusqu'à son origine. Non retenu.

## 13. Ordre d'implémentation

1. ~~**Modèle et stockage**~~ — fait : `model/activity_event.go`,
   `model/activity_payload.go`, `database.Models`, les deux FK dans
   `tools/atlas-loader/main.go`, migration Atlas, `auth/usercontext/`,
   `repository/activity_event_repository.go`, `service/activity_service.go`,
   `repositories.go` / `services.go`, `bin/mock.sh`, tests.
2. ~~**Émission**~~ — fait : les 4 services de §6.2, `ImportRedirectOptions.Filename`
   renseigné par le resolver d'import, reprise des tests existants et tests
   d'émission dédiés.
3. ~~**API**~~ — fait : `graph/schema/activity.graphqls`, `gqlgen.yml`,
   `gqlgen generate`, `graph/resolver/activity.resolvers.go`, injection de
   `ActivityService` dans `graph/resolver/resolver.go` et `http/server.go`. Vérifié de
   bout en bout sur MariaDB : émission, filtres, et refus de permission.
4. ~~**Webui**~~ — fait : §10, plus `ResourceType.Any` ajouté à `usePermissions`
   (le front n'avait pas le joker « any » du backend) et les titres manquants du
   `Header` (`activity`, mais aussi `agents` et `redirect-tester`, qui retombaient
   déjà à tort sur « Dashboard »). Vérifié dans Chrome sur une instance jetable :
   rendu des 6 types d'événement, fallback générique sur un type inconnu, diff
   limité aux champs modifiés, troncature des chemins, filtres, mode sombre.
5. ~~**Purge et scheduler**~~ — fait : `scheduler/`, `scheduler/task/activity_purge.go`,
   `config.ActivityConfig`, `ActivityService.Purge`, les trois primitives du
   repository, la commande `db activity-purge`, et le câblage dans `cli/start.go`.
   Vérifié sur MariaDB : 2041 événements ramenés à 10 en gardant les plus récents,
   purge au démarrage, purge sur tick, arrêt propre sur SIGTERM.
6. **Documentation** — `.claude/CLAUDE.md` (structure, config, modèle de données) et
   `docs/docs/` (page fonctionnelle + section configuration). À noter : le workflow
   de migration Atlas (`bin/atlas-diff.sh`, `tools/atlas-loader`) n'est documenté
   nulle part aujourd'hui — bon moment pour l'ajouter.

## 14. Points ouverts

1. **Plafond par défaut** : 1000 events par projet. C'est le seul garde-fou, et il
   détermine aussi la profondeur d'historique visible dans le webui — à confirmer
   au regard du rythme réel d'un projet actif (un import = 1 event, mais une
   automatisation par token API peut en produire beaucoup).
2. **FK utilisateur en base ou `UPDATE` applicatif** (§3.3) : la FK
   `ON DELETE SET NULL` est plus sûre mais non testable en unitaire. Retenu : la FK.