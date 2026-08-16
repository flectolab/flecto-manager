-- modify "pages" table
ALTER TABLE `pages` ADD INDEX `idx_pages_ns_proj_created` (`namespace_code`, `project_code`, `created_at` DESC), ADD INDEX `idx_pages_ns_proj_type` (`namespace_code`, `project_code`, `type`), ADD INDEX `idx_pages_ns_proj_updated` (`namespace_code`, `project_code`, `updated_at` DESC);
-- modify "redirects" table
ALTER TABLE `redirects` ADD INDEX `idx_redirects_ns_proj_created` (`namespace_code`, `project_code`, `created_at` DESC), ADD INDEX `idx_redirects_ns_proj_type` (`namespace_code`, `project_code`, `type`), ADD INDEX `idx_redirects_ns_proj_updated` (`namespace_code`, `project_code`, `updated_at` DESC);
