-- create "roles" table
CREATE TABLE `roles` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `code` varchar(100) NOT NULL,
  `type` varchar(100) NOT NULL,
  `created_at` timestamp NULL,
  `updated_at` timestamp NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_role_code_type` (`code`, `type`)
) COLLATE utf8mb4_uca1400_ai_ci;
-- create "tokens" table
CREATE TABLE `tokens` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(300) NOT NULL,
  `token_hash` varchar(64) NOT NULL,
  `token_preview` varchar(30) NOT NULL,
  `expires_at` timestamp NULL,
  `created_at` timestamp NULL,
  `updated_at` timestamp NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_tokens_name` (`name`),
  UNIQUE INDEX `idx_tokens_token_hash` (`token_hash`)
) COLLATE utf8mb4_uca1400_ai_ci;
-- create "admin_permissions" table
CREATE TABLE `admin_permissions` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `section` varchar(100) NOT NULL,
  `action` varchar(50) NOT NULL,
  `role_id` bigint NULL,
  `created_at` timestamp NULL,
  PRIMARY KEY (`id`),
  INDEX `fk_roles_admin` (`role_id`),
  INDEX `idx_admin_perm_section` (`section`),
  CONSTRAINT `fk_roles_admin` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`) ON UPDATE RESTRICT ON DELETE CASCADE
) COLLATE utf8mb4_uca1400_ai_ci;
-- create "namespaces" table
CREATE TABLE `namespaces` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `namespace_code` varchar(50) NULL,
  `name` longtext NULL,
  `created_at` timestamp NULL,
  `updated_at` timestamp NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_namespace_namespace_code` (`namespace_code`)
) COLLATE utf8mb4_uca1400_ai_ci;
-- create "projects" table
CREATE TABLE `projects` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `project_code` varchar(50) NULL,
  `namespace_code` varchar(50) NULL,
  `name` longtext NULL,
  `version` bigint NULL DEFAULT 1,
  `created_at` timestamp NULL,
  `updated_at` timestamp NULL,
  `published_at` timestamp NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_namespace` (`namespace_code`),
  UNIQUE INDEX `idx_projects_namespace_project` (`namespace_code`, `project_code`),
  UNIQUE INDEX `idx_project_namespace` (`project_code`, `namespace_code`),
  CONSTRAINT `fk_projects_namespace` FOREIGN KEY (`namespace_code`) REFERENCES `namespaces` (`namespace_code`) ON UPDATE RESTRICT ON DELETE CASCADE
) COLLATE utf8mb4_uca1400_ai_ci;
-- create "agents" table
CREATE TABLE `agents` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `namespace_code` varchar(50) NULL,
  `project_code` varchar(50) NULL,
  `name` varchar(100) NULL,
  `status` varchar(50) NULL,
  `type` varchar(50) NULL,
  `version` bigint NULL,
  `load_duration` bigint NULL,
  `error` varchar(500) NULL,
  `created_at` timestamp NULL,
  `updated_at` timestamp NULL,
  `last_hit_at` timestamp NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_agents_namespace_project_name` (`namespace_code`, `project_code`, `name`),
  INDEX `idx_pages_namespace_project` (`namespace_code`, `project_code`),
  CONSTRAINT `fk_agents_project` FOREIGN KEY (`namespace_code`, `project_code`) REFERENCES `projects` (`namespace_code`, `project_code`) ON UPDATE RESTRICT ON DELETE CASCADE
) COLLATE utf8mb4_uca1400_ai_ci;
-- create "pages" table
CREATE TABLE `pages` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `namespace_code` varchar(50) NULL,
  `project_code` varchar(50) NULL,
  `is_published` bool NOT NULL DEFAULT 0,
  `published_at` timestamp NULL,
  `content_size` bigint NOT NULL DEFAULT 0,
  `type` varchar(50) NULL,
  `path` varchar(600) NULL,
  `content` longtext NULL,
  `content_type` varchar(50) NULL,
  `created_at` timestamp NULL,
  `updated_at` timestamp NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_pages_namespace_project` (`namespace_code`, `project_code`),
  UNIQUE INDEX `idx_pages_path_unique` (`namespace_code`, `project_code`, `path`),
  CONSTRAINT `fk_pages_project` FOREIGN KEY (`namespace_code`, `project_code`) REFERENCES `projects` (`namespace_code`, `project_code`) ON UPDATE RESTRICT ON DELETE CASCADE
) COLLATE utf8mb4_uca1400_ai_ci;
-- create "page_drafts" table
CREATE TABLE `page_drafts` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `namespace_code` varchar(50) NULL,
  `project_code` varchar(50) NULL,
  `change_type` varchar(50) NULL,
  `old_page_id` bigint NULL,
  `content_size` bigint NOT NULL DEFAULT 0,
  `new_type` varchar(50) NULL,
  `new_path` varchar(600) NULL,
  `new_content` longtext NULL,
  `new_content_type` varchar(50) NULL,
  `created_at` timestamp NULL,
  `updated_at` timestamp NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_page_drafts_namespace_project` (`namespace_code`, `project_code`),
  INDEX `idx_page_drafts_old_page_id` (`old_page_id`),
  UNIQUE INDEX `idx_page_drafts_path_unique` (`namespace_code`, `project_code`, `new_path`),
  CONSTRAINT `fk_page_drafts_project` FOREIGN KEY (`namespace_code`, `project_code`) REFERENCES `projects` (`namespace_code`, `project_code`) ON UPDATE RESTRICT ON DELETE CASCADE,
  CONSTRAINT `fk_pages_page_draft` FOREIGN KEY (`old_page_id`) REFERENCES `pages` (`id`) ON UPDATE RESTRICT ON DELETE CASCADE
) COLLATE utf8mb4_uca1400_ai_ci;
-- create "redirects" table
CREATE TABLE `redirects` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `namespace_code` varchar(50) NULL,
  `project_code` varchar(50) NULL,
  `is_published` bool NOT NULL DEFAULT 0,
  `published_at` timestamp NULL,
  `type` varchar(50) NULL,
  `source` varchar(600) NULL,
  `target` varchar(2048) NULL,
  `status` varchar(50) NULL,
  `created_at` timestamp NULL,
  `updated_at` timestamp NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_redirects_namespace_project` (`namespace_code`, `project_code`),
  UNIQUE INDEX `idx_redirects_source_unique` (`namespace_code`, `project_code`, `source`),
  CONSTRAINT `fk_redirects_project` FOREIGN KEY (`namespace_code`, `project_code`) REFERENCES `projects` (`namespace_code`, `project_code`) ON UPDATE RESTRICT ON DELETE CASCADE
) COLLATE utf8mb4_uca1400_ai_ci;
-- create "redirect_drafts" table
CREATE TABLE `redirect_drafts` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `namespace_code` varchar(50) NULL,
  `project_code` varchar(50) NULL,
  `change_type` varchar(50) NULL,
  `old_redirect_id` bigint NULL,
  `new_type` varchar(50) NULL,
  `new_source` varchar(600) NULL,
  `new_target` varchar(2048) NULL,
  `new_status` varchar(50) NULL,
  `created_at` timestamp NULL,
  `updated_at` timestamp NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_redirect_drafts_namespace_project` (`namespace_code`, `project_code`),
  INDEX `idx_redirect_drafts_old_redirect_id` (`old_redirect_id`),
  UNIQUE INDEX `idx_redirect_drafts_source_unique` (`namespace_code`, `project_code`, `new_source`),
  CONSTRAINT `fk_redirect_drafts_project` FOREIGN KEY (`namespace_code`, `project_code`) REFERENCES `projects` (`namespace_code`, `project_code`) ON UPDATE RESTRICT ON DELETE CASCADE,
  CONSTRAINT `fk_redirects_redirect_draft` FOREIGN KEY (`old_redirect_id`) REFERENCES `redirects` (`id`) ON UPDATE RESTRICT ON DELETE CASCADE
) COLLATE utf8mb4_uca1400_ai_ci;
-- create "resource_permissions" table
CREATE TABLE `resource_permissions` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `namespace` varchar(50) NOT NULL,
  `project` varchar(50) NULL,
  `resource` varchar(50) NOT NULL,
  `action` varchar(50) NOT NULL,
  `role_id` bigint NULL,
  `created_at` timestamp NULL,
  PRIMARY KEY (`id`),
  INDEX `fk_roles_resources` (`role_id`),
  INDEX `idx_res_perm_namespace` (`namespace`),
  INDEX `idx_res_perm_project` (`project`),
  CONSTRAINT `fk_roles_resources` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`) ON UPDATE RESTRICT ON DELETE CASCADE
) COLLATE utf8mb4_uca1400_ai_ci;
-- create "users" table
CREATE TABLE `users` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `username` varchar(100) NOT NULL,
  `password` varchar(255) NULL,
  `lastname` longtext NULL,
  `firstname` longtext NULL,
  `active` bool NOT NULL DEFAULT 1,
  `refresh_token_hash` varchar(255) NULL,
  `created_at` timestamp NULL,
  `updated_at` timestamp NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `uni_users_username` (`username`)
) COLLATE utf8mb4_uca1400_ai_ci;
-- create "user_roles" table
CREATE TABLE `user_roles` (
  `user_id` bigint NOT NULL,
  `role_id` bigint NOT NULL,
  `created_at` timestamp NULL,
  PRIMARY KEY (`user_id`, `role_id`),
  INDEX `fk_user_roles_role` (`role_id`),
  CONSTRAINT `fk_user_roles_role` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`) ON UPDATE RESTRICT ON DELETE CASCADE,
  CONSTRAINT `fk_user_roles_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON UPDATE RESTRICT ON DELETE CASCADE
) COLLATE utf8mb4_uca1400_ai_ci;
