-- create "activity_events" table
CREATE TABLE `activity_events` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `namespace_code` varchar(50) NOT NULL,
  `project_code` varchar(50) NOT NULL,
  `resource` varchar(20) NOT NULL,
  `action` varchar(20) NOT NULL,
  `user_id` bigint NULL,
  `actor` varchar(300) NOT NULL,
  `auth_type` varchar(20) NULL,
  `resource_id` bigint NULL,
  `data` longtext NULL,
  `occurred_at` timestamp NOT NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_activity_events_ns_proj` (`namespace_code`, `project_code`),
  INDEX `idx_activity_events_resource` (`resource_id`),
  INDEX `idx_activity_events_user` (`user_id`),
  CONSTRAINT `fk_activity_events_project` FOREIGN KEY (`namespace_code`, `project_code`) REFERENCES `projects` (`namespace_code`, `project_code`) ON UPDATE RESTRICT ON DELETE CASCADE,
  CONSTRAINT `fk_activity_events_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON UPDATE RESTRICT ON DELETE SET NULL
) COLLATE utf8mb4_uca1400_ai_ci;
