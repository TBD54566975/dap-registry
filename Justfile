set positional-arguments

_help:
  @just -l


build:
  #!/bin/bash
  set -euo pipefail

  ftl build backend/modules

bump-ftl:
  #!/bin/bash
  set -euo pipefail

  hermit update
  hermit upgrade ftl
  cd backend && go get github.com/TBD54566975/ftl@latest && cd ..
  just tidy

# Bump the Go version and update all go.mod files.
bump-go:
  #!/bin/bash
  set -euo pipefail

  GO_VERSION=$(go version | awk '{print $3}' | cut -c 3-)
  echo $GO_VERSION

  hermit upgrade go
  GO_VERSION=$(go version | awk '{print $3}' | cut -c 3-)
  echo $GO_VERSION

  find backend -name go.mod | grep -v /_ | xargs -n1 dirname | xargs -n1 -I{} sh -c "echo 'updating {}'; cd {} && go mod edit -go=$GO_VERSION && go mod tidy"

# create a new migration file for the given backend module and table.
dbmate-new module action table:
  #!/bin/bash
  set -euo pipefail
  
  mkdir backend/modules/{{module}}/migrations || true
  
  dbmate --migrations-dir backend/modules/{{module}}/migrations new {{action}}_{{table}}

# Run the migrations for a module.
dbmate-up module $drop="":
  #!/bin/bash
  set -euo pipefail
  
  migrations_dir=backend/modules/{{module}}/migrations
  module_db={{uppercase(module)}}_{{uppercase(module)}}
  
  dbsec="{{module}}.FTL_DSN_$module_db"
  test_dbsec="{{module}}.FTL_TEST_DSN_$module_db"

  # Handle the main database URL
  if dbsec=$(ftl secret get "$dbsec" 2>/dev/null); then
    dburl=$(echo "$dbsec" | jq -r)

    if [ "$drop" == "drop" ]; then
      dbmate --url "$dburl" --migrations-dir "$migrations_dir" drop || true
    fi
    
    dbmate --url "$dburl" --migrations-dir "$migrations_dir" up
  else
    echo "Main database URL not found for $dbsec"
  fi

  # Handle the test database URL
  if testdbsec=$(ftl secret get "$test_dbsec" 2>/dev/null); then
    testdburl=$(echo "$testdbsec" | jq -r)
    dbmate --url "$testdburl" --migrations-dir "$migrations_dir" drop || true
    dbmate --url "$testdburl" --migrations-dir "$migrations_dir" up
  else
    echo "Test database URL not found for $test_dbsec"
  fi

# Run the registry in dev mode:
dev:
  #!/bin/bash
  set -euo pipefail

  export FTL_CONFIG="$(pwd)/ftl-project.toml"
  ftl dev --recreate -j2 --bind=http://0.0.0.0:8891 --log-timestamps --allow-origins '*'


didweb domain="http://localhost:8892/ingress":
  #!/bin/bash
  set -euo pipefail

  echo -n $(cd backend; go run cmd/didweb/didweb.go {{domain}}) | ftl secret set -C ftl-project.toml --inline  did_web_portable_did

# Run schema migrations for all modules.
migrate $drop="":
  #!/bin/bash
  set -euo pipefail

  # find all the modules with migrations
  modules=$(ls -d1 backend/modules/*/migrations | cut -d/ -f3)

  for module in $modules; do
    if ! just dbmate-up $module $drop; then
      echo "Failed to run dbmate-up for module $module"
      exit 1
    fi
  done

# scaffold a new FTL module
module module:
  #!/bin/bash
  set -euo pipefail
  
  ftl init go backend/modules --replace github.com/TBD54566975/dap-registry/backend/modules=../.. {{module}}
  git add backend/modules/{{module}}

# scaffold a new FTL module with a DB 
module-with-db module:
  #!/bin/bash
  set -euo pipefail
  
  # Reuse the module recipe
  just module {{module}}
  
  # Add database configuration
  echo -n "postgres://postgres:secret@localhost:15432/{{module}}_{{module}}?sslmode=disable" | ftl secret set --inline -C ftl-project.toml {{module}}.FTL_DSN_{{uppercase(module)}}_{{uppercase(module)}}
  echo -n "postgres://postgres:secret@localhost:15432/{{module}}_{{module}}_test?sslmode=disable" | ftl secret set --inline -C ftl-project.toml {{module}}.FTL_TEST_DSN_{{uppercase(module)}}_{{uppercase(module)}}

test:
  #!/bin/bash
  set -euo pipefail

  just migrate
  just build

  dirs=("$@")
  if [ ${#dirs[@]} -eq 0 ]; then
    dirs=($(find backend -name go.mod | grep -v /_ | xargs -n1 dirname))
  fi


  for dir in "${dirs[@]}"; do
    echo "Testing $dir"
    (cd "$dir" && go test -fullpath -v -p 1 ./...)
  done

test-ci:
  #!/bin/bash
  set -euo pipefail

  trap "ftl serve --stop" EXIT ERR INT
  ftl serve --recreate --background --stop
  just test

# run go mod tidy for all modules
tidy:
  #!/bin/bash
  set -euo pipefail
  
  find backend -name go.mod | grep -v /_ | xargs -n1 dirname | xargs -n1 -I{} sh -c "echo 'updating {}'; cd {} && go mod tidy"