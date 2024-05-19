set positional-arguments

_help:
  @just -l

dbmate-up module:
  #!/bin/bash
  set -euo pipefail
  
  migrations_dir=backend/modules/{{module}}/migrations
  
  dburl=`ftl secret get -C ftl-project.toml {{module}}.FTL_DSN_{{uppercase(module)}}_{{uppercase(module)}} | jq -r`
  testdburl=`ftl secret get -C ftl-project.toml {{module}}.FTL_TEST_DSN_{{uppercase(module)}}_{{uppercase(module)}} | jq -r`

  dbmate --url $testdburl --migrations-dir $migrations_dir drop || true
  dbmate --url $testdburl --migrations-dir $migrations_dir up

  dbmate --url $dburl --migrations-dir $migrations_dir drop || true
  dbmate --url $dburl --migrations-dir $migrations_dir up

# Create a new migration file for the given backend module and table.
dbmate-new module action table:
  #!/bin/bash
  set -euo pipefail
  
  mkdir backend/modules/{{module}}/migrations || true
  dbmate --migrations-dir backend/modules/{{module}}/migrations new {{action}}_{{table}}


didweb domain="http://localhost:8892/ingress":
  #!/bin/bash
  set -euo pipefail

  echo -n $(cd backend; go run cmd/didweb/didweb.go {{domain}}) | ftl secret set -C ftl-project.toml --inline  did_web_portable_did

module module:
  #!/bin/bash
  set -euo pipefail
  
  ftl init go backend/modules --replace github.com/TBD54566975/dapregistry/backend/modules=../.. {{module}}
  git add backend/modules/{{module}}

# Scaffold a new FTL module with a DB.
module-with-db module:
  #!/bin/bash
  set -euo pipefail
  
  ftl init go backend/modules --replace github.com/TBD54566975/dapregistry/backend/modules=../.. {{module}}
  echo -n "postgres://postgres:secret@localhost:54320/{{module}}_{{module}}?sslmode=disable" | ftl secret set --inline -C ftl-project.toml {{module}}.FTL_DSN_{{uppercase(module)}}_{{uppercase(module)}}
  echo -n "postgres://postgres:secret@localhost:54320/{{module}}_{{module}}_test?sslmode=disable" | ftl secret set --inline -C ftl-project.toml {{module}}.FTL_TEST_DSN_{{uppercase(module)}}_{{uppercase(module)}}

# run go mod tidy in all modules
tidy:
  #!/bin/bash
  set -euo pipefail
  
  find backend -name go.mod | grep -v /_ | xargs -n1 dirname | xargs -n1 -I{} sh -c "echo 'updating {}'; cd {} && go mod tidy"

