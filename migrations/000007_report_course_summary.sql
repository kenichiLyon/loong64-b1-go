ALTER TABLE report_exports
  DROP CONSTRAINT IF EXISTS report_exports_report_type_check,
  ADD CONSTRAINT report_exports_report_type_check
    CHECK (report_type IN ('submission_report', 'experiment_summary', 'course_summary'));

ALTER TABLE report_exports
  DROP CONSTRAINT IF EXISTS report_exports_scope_type_check,
  ADD CONSTRAINT report_exports_scope_type_check
    CHECK (scope_type IN ('submission', 'experiment', 'course'));
