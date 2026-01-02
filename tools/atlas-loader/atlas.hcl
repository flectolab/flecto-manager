data "external_schema" "gorm" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "./tools/atlas-loader",
  ]
}

env "local" {
  src = data.external_schema.gorm.url
  dev = "docker://mariadb/latest/dev"
  migration {
    dir    = "file://migrations"
    format = golang-migrate
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
