# Database migration guidance

- Keep this Go module independently buildable and outside Padinho's container
  image, Compose lifecycle, and GitHub Actions workflow.
- Embed every ordered SQL migration in the Linux migration executable so an
  operator only needs to upload one artifact to the Padinho VM.
- Apply migrations manually from the Padinho VM, which is the only host allowed
  to reach private MySQL. Never apply schema changes automatically after a push.
- Keep schema creation and migration execution in this module. Applications may
  consume the resulting schema but must not create or migrate it.
- Test migration changes against live MySQL through the `TEST_DATABASE_*`
  environment contract in addition to ordinary unit tests.
