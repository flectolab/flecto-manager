-- modify "pages" table
ALTER TABLE `pages` DROP INDEX `idx_pages_namespace_project`;
-- modify "pages" table
ALTER TABLE `pages` ADD INDEX `idx_pages_namespace_project` (`namespace_code`, `project_code`, `is_published`);
-- modify "redirects" table
ALTER TABLE `redirects` DROP INDEX `idx_redirects_namespace_project`;
-- modify "redirects" table
ALTER TABLE `redirects` ADD INDEX `idx_redirects_namespace_project` (`namespace_code`, `project_code`, `is_published`);
