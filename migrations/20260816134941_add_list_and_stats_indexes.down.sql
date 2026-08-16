-- reverse: modify "redirects" table
ALTER TABLE `redirects` DROP INDEX `idx_redirects_ns_proj_updated`, DROP INDEX `idx_redirects_ns_proj_type`, DROP INDEX `idx_redirects_ns_proj_created`;
-- reverse: modify "pages" table
ALTER TABLE `pages` DROP INDEX `idx_pages_ns_proj_updated`, DROP INDEX `idx_pages_ns_proj_type`, DROP INDEX `idx_pages_ns_proj_created`;
