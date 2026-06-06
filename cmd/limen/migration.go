package main

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/thecodearcher/limen"
)

type Migration struct {
	Version string // Migration version timestamp (YYYYMMDDHHMMSS)
	UpSQL   string // SQL to apply the migration
	DownSQL string // SQL to rollback the migration
}

func generateMigrations(db *sql.DB, driver Driver, config *cliConfig) ([]Migration, error) {
	migrations := make([]Migration, 0, len(config.Schemas))
	timestamp := time.Now()

	introspector := newSchemaIntrospector(db, driver)
	tableNames := make([]string, 0, len(config.Schemas))
	for _, schema := range config.Schemas {
		tableNames = append(tableNames, string(schema.GetTableName()))
	}

	existingTables, err := introspector.getTables(tableNames)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing tables: %w", err)
	}

	generator, err := newSQLMigrationGenerator(driver, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create migration generator: %w", err)
	}

	schemaNames, err := orderSchemasByDependencies(config.Schemas)
	if err != nil {
		return nil, err
	}

	for _, schemaName := range schemaNames {
		schemaDef := config.Schemas[schemaName]
		var diff *schemaDiff

		if existingTables[string(schemaDef.GetTableName())] {
			diff, err = generateDiffForTable(introspector, &schemaDef)
			if err != nil {
				return nil, fmt.Errorf("failed to generate schema diff for %s: %w", schemaName, err)
			}

			if !diff.HasChanges() {
				continue
			}
		}

		upSQL, err := generator.generateUpMigration(&schemaDef, diff)
		if err != nil {
			return nil, fmt.Errorf("failed to generate up migration for %s: %w", schemaName, err)
		}

		downSQL, err := generator.generateDownMigration(&schemaDef, diff)
		if err != nil {
			return nil, fmt.Errorf("failed to generate down migration for %s: %w", schemaName, err)
		}

		migration := Migration{
			Version: migrationVersion(timestamp, len(migrations), schemaName),
			UpSQL:   upSQL,
			DownSQL: downSQL,
		}

		migrations = append(migrations, migration)
	}

	return migrations, nil
}

func migrationVersion(base time.Time, sequence int, schemaName limen.SchemaName) string {
	return fmt.Sprintf("%s_%s", base.Add(time.Duration(sequence)*time.Second).Format("20060102150405"), schemaName)
}

func orderSchemasByDependencies(schemas limen.SchemaDefinitionMap) ([]limen.SchemaName, error) {
	names := sortedSchemaNames(schemas)
	tableToSchema := make(map[limen.SchemaName]limen.SchemaName, len(schemas))
	for _, name := range names {
		schema := schemas[name]
		tableToSchema[limen.SchemaName(schema.GetTableName())] = name
	}

	indegree := make(map[limen.SchemaName]int, len(schemas))
	dependents := make(map[limen.SchemaName][]limen.SchemaName, len(schemas))
	seenEdges := make(map[limen.SchemaName]map[limen.SchemaName]bool, len(schemas))
	for _, name := range names {
		indegree[name] = 0
	}

	for _, name := range names {
		for _, fk := range schemas[name].ForeignKeys {
			dependency, ok := resolveSchemaDependency(fk.ReferencedSchema, schemas, tableToSchema)
			if !ok || dependency == name {
				continue
			}
			if seenEdges[name] == nil {
				seenEdges[name] = make(map[limen.SchemaName]bool)
			}
			if seenEdges[name][dependency] {
				continue
			}

			seenEdges[name][dependency] = true
			indegree[name]++
			dependents[dependency] = append(dependents[dependency], name)
		}
	}

	ready := make([]limen.SchemaName, 0, len(schemas))
	for _, name := range names {
		if indegree[name] == 0 {
			ready = append(ready, name)
		}
	}

	ordered := make([]limen.SchemaName, 0, len(schemas))
	for len(ready) > 0 {
		sortSchemaNames(ready)
		name := ready[0]
		ready = ready[1:]
		ordered = append(ordered, name)

		for _, dependent := range dependents[name] {
			indegree[dependent]--
			if indegree[dependent] == 0 {
				ready = append(ready, dependent)
			}
		}
	}

	if len(ordered) != len(schemas) {
		return nil, fmt.Errorf("foreign key dependency cycle detected among schemas")
	}

	return ordered, nil
}

func resolveSchemaDependency(
	referenced limen.SchemaName,
	schemas limen.SchemaDefinitionMap,
	tableToSchema map[limen.SchemaName]limen.SchemaName,
) (limen.SchemaName, bool) {
	if _, ok := schemas[referenced]; ok {
		return referenced, true
	}
	name, ok := tableToSchema[referenced]
	return name, ok
}

func sortedSchemaNames(schemas limen.SchemaDefinitionMap) []limen.SchemaName {
	names := make([]limen.SchemaName, 0, len(schemas))
	for name := range schemas {
		names = append(names, name)
	}
	sortSchemaNames(names)
	return names
}

func sortSchemaNames(names []limen.SchemaName) {
	sort.Slice(names, func(i, j int) bool {
		return string(names[i]) < string(names[j])
	})
}

func generateDiffForTable(introspector *schemaIntrospector, schema *limen.SchemaDefinition) (*schemaDiff, error) {
	existingSchema, err := introspector.introspectTable(schema.GetTableName())
	if err != nil {
		return nil, err
	}
	diff := compareSchemas(existingSchema, schema)
	return &diff, nil
}
