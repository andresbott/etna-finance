# Category Rule Groups Design

## Problem

The current category matching system uses a flat list of rules. Each rule has one pattern, one category, and a position. As the number of patterns grows, the list becomes unwieldy. Many rules share the same target category and logically belong together.

## Solution

Replace the flat rule list with **rule groups**. Each group has a name, a target category, a position (for evaluation order), and N patterns. Each pattern is independently either plain text (substring match) or regex.

Multiple groups can point to the same category.

## Data Model

### CategoryRuleGroup (new table: `category_rule_groups`)

| Field      | Type      | Notes                         |
|------------|-----------|-------------------------------|
| ID         | uint      | primary key                   |
| Name       | string    | user-defined label, required  |
| CategoryID | uint      | target category, required     |
| Position   | int       | group evaluation order        |
| CreatedAt  | time.Time |                               |
| UpdatedAt  | time.Time |                               |

### CategoryRulePattern (new table: `category_rule_patterns`)

| Field     | Type      | Notes                        |
|-----------|-----------|------------------------------|
| ID        | uint      | primary key                  |
| GroupID   | uint      | FK to CategoryRuleGroup      |
| Pattern   | string    | match string, required       |
| IsRegex   | bool      | regex or plain text          |
| CreatedAt | time.Time |                              |
| UpdatedAt | time.Time |                              |

No Position or CategoryID on patterns. Those live on the group.

### GORM relationships

The Go struct for `dbCategoryRuleGroup` needs a `Patterns []dbCategoryRulePattern` field with `gorm:"foreignKey:GroupID"` to support `Preload("Patterns")` when loading groups with nested patterns. This is the first has-many relationship in this package.

### Migration

Each existing `CategoryRule` becomes a group (Name = Pattern value, CategoryID preserved, Position preserved) with one child pattern. The old `db_category_rules` table is dropped after migration.

The migration runs as a Go function at startup (since GORM AutoMigrate can create new tables but cannot drop old ones or migrate data). Steps:
1. AutoMigrate the new tables
2. Read existing `db_category_rules` rows
3. Create corresponding groups + patterns
4. Drop the old `db_category_rules` table

## Matching Logic

```
MatchCategory(description, groups):
  for each group (ordered by Position ASC, ID ASC):
    for each pattern in group:
      if pattern.IsRegex -> regex match
      else -> case-insensitive substring match
    if any pattern matched -> return group.CategoryID
  return 0
```

First matching group wins. Pattern order within a group is irrelevant.

The `Parse()` function signature also changes from `rules []CategoryRule` to `groups []CategoryRuleGroup`, since it passes them to `MatchCategory`.

## API

### Removed

- `GET /import/category-rules`
- `POST /import/category-rules`
- `PUT /import/category-rules/:id`
- `DELETE /import/category-rules/:id`

### Added

**Rule Groups:**

- `GET /import/category-rule-groups` -- returns groups with nested patterns, ordered by position
- `POST /import/category-rule-groups` -- create group (with optional initial patterns)
- `PUT /import/category-rule-groups/:id` -- update group (name, categoryId, position)
- `DELETE /import/category-rule-groups/:id` -- deletes group and all its patterns (delete patterns first, then group)

**Patterns (nested under group):**

- `POST /import/category-rule-groups/:groupId/patterns` -- add pattern to group
- `PUT /import/category-rule-groups/:groupId/patterns/:id` -- update pattern
- `DELETE /import/category-rule-groups/:groupId/patterns/:id` -- delete pattern

Pattern create/update endpoints must validate regex patterns (compile check) when `isRegex` is true, same as the current rule validation.

The nested routes require extracting both `{groupId}` and `{id}` from the URL. The existing `getId()` helper only extracts `{id}`, so pattern routes need additional extraction for `groupId`.

### Response format (list)

```json
[
  {
    "id": 1,
    "name": "Amazon purchases",
    "categoryId": 5,
    "position": 0,
    "patterns": [
      { "id": 1, "pattern": "AMAZON", "isRegex": false },
      { "id": 2, "pattern": "AMZN.*MKTP", "isRegex": true }
    ]
  }
]
```

## Backup/Restore

Update the V1 schema. The `category_rules.json` file changes to the grouped format. No backward compatibility with old flat format.

**New `category_rules.json` format:**

```json
[
  {
    "id": 1,
    "name": "Amazon purchases",
    "categoryId": 5,
    "position": 0,
    "patterns": [
      { "id": 1, "pattern": "AMAZON", "isRegex": false },
      { "id": 2, "pattern": "AMZN.*MKTP", "isRegex": true }
    ]
  }
]
```

## Frontend

### TypeScript types

```ts
interface CategoryRuleGroup {
  id: number
  name: string
  categoryId: number
  position: number
  patterns: CategoryRulePattern[]
}

interface CategoryRulePattern {
  id: number
  pattern: string
  isRegex: boolean
}
```

The old `CategoryRule` type is removed.

### CategoryRulesView.vue

- Main view: ordered list of groups using PrimeVue DataTable with row expansion
  - Columns: Position, Name, Category, # Patterns, Actions (edit/delete)
  - Group create/edit happens in a dialog (name, category, position)
- Expanded row: list of patterns within the group
  - Columns: Pattern, Type (Regex/Substring), Actions (edit/delete)
  - "Add Pattern" button within expanded row

## Files to modify

| Layer        | File                                              | Change                                              |
|--------------|---------------------------------------------------|-----------------------------------------------------|
| Store init   | `internal/csvimport/csvimport.go`                 | Update AutoMigrate + WipeData for new tables        |
| Store        | `internal/csvimport/category_rule.go`             | Replace with group + pattern models and CRUD        |
| Store test   | `internal/csvimport/category_rule_test.go`        | Rewrite for new models                              |
| Store test   | `internal/csvimport/wipe_test.go`                 | Update to use new model/method names                |
| Parser       | `internal/csvimport/parser.go`                    | Update MatchCategory + Parse to accept groups       |
| Parser test  | `internal/csvimport/parser_test.go`               | Update matching tests                               |
| Handler      | `app/router/handlers/csvimport/category_rule.go`  | Replace with group + pattern handlers               |
| Handler      | `app/router/handlers/csvimport/import.go`         | Use ListCategoryRuleGroups + pass groups to Parse   |
| Router       | `app/router/api_v0.go`                            | Update route registration                           |
| Reapply      | `app/router/handlers/csvimport/reapply.go`        | Use new ListCategoryRuleGroups + pass groups        |
| Reapply test | `app/router/handlers/csvimport/reapply_test.go`   | Update if needed                                    |
| Backup data  | `internal/backup/dataV1.go`                       | Replace categoryRuleV1 with grouped struct          |
| Backup export| `internal/backup/export.go`                       | Write grouped format                                |
| Backup import| `internal/backup/import.go`                       | Read grouped format + update loadV1Json type union  |
| Backup tests | `internal/backup/export_test.go`, `import_test.go`| Update for new format                               |
| TS types     | `webui/src/types/csvimport.ts`                    | Replace CategoryRule with group + pattern types     |
| API client   | `webui/src/lib/api/CsvImport.ts`                  | New API functions for groups + patterns             |
| API test     | `webui/src/lib/api/CsvImport.test.ts`             | Update                                              |
| View         | `webui/src/views/csvimport/CategoryRulesView.vue` | Grouped list with expandable rows + dialogs         |
