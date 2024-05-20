{ pkgs ? (
    let
      inherit (builtins) fetchTree fromJSON readFile;
    in
    import (fetchTree (fromJSON (readFile ./flake.lock)).nodes.nixpkgs.locked) {
      overlays = [
        (import ./overlay.nix)
      ];
    }
  )
}:

pkgs.mkShell {
  nativeBuildInputs = [
    pkgs.nixpkgs-fmt
    pkgs.golangci-lint
    pkgs.gomod2nix
    (pkgs.mkGoEnv { pwd = ./.; })
  ];
}
