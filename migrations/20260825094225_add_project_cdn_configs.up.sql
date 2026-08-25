-- create "project_cdn_configs" table
CREATE TABLE `project_cdn_configs` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `namespace_code` varchar(50) NULL,
  `project_code` varchar(50) NULL,
  `name` varchar(100) NOT NULL,
  `type` varchar(50) NOT NULL,
  `enabled` bool NOT NULL DEFAULT 1,
  `settings` longtext NULL,
  `secrets` longtext NULL,
  `created_at` timestamp NULL,
  `updated_at` timestamp NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_project_cdn_configs_name` (`namespace_code`, `project_code`, `name`),
  CONSTRAINT `fk_project_cdn_configs_project` FOREIGN KEY (`namespace_code`, `project_code`) REFERENCES `projects` (`namespace_code`, `project_code`) ON UPDATE RESTRICT ON DELETE CASCADE
) COLLATE utf8mb4_uca1400_ai_ci;
