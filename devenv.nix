{ pkgs, lib, config, ... }: {
  languages.go.enable = true;

  packages = [
    pkgs.duckdb
  ];
  # See full reference at https://devenv.sh/reference/options/
}
