#! /bin/sh

set -e

db_file_path=${DB_FILE_PATH:-elapsed.db}

if [ ! -f "${db_file_path}" ]; then
  ./scripts/create_db
fi

./main
