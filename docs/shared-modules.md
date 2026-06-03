# Shared Module Plans

AppSuite does not introduce shared modules yet. ModuleCRM owns contacts and
activities for the standalone CRM product today. Shared modules become justified
only when at least two product modules need the same record ownership.

## Shared Contacts

The future shared contacts module should own people, companies, communication
channels, and deduplication rules when CMS, CRM, Support, and Ideas all need the
same contact graph.

Initial extraction trigger:

- CRM and Support both need contacts with independent panel surfaces.
- CMS authors or subscribers need to relate to the same people records.
- Cross-workspace relation policies require one audited source of truth.

Until then, ModuleCRM remains the owner for contacts in CRM and Sales spaces.

## Shared Activity Timeline

The future shared activity timeline module should own events, notes, comments,
task changes, audit summaries, and workspace-aware timeline projections.

Initial extraction trigger:

- CRM activities and Support ticket updates need a shared chronological model.
- Ideas, SEO, and Optimize need to attach timeline events to their own records.
- Cross-workspace links need explicit capability-gated read models.

Until then, ModuleCRM remains the owner for activities and notes in CRM spaces.
