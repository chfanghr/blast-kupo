{
  pkgs ? (
    let
      inherit (builtins) fetchTree fromJSON readFile;
      inherit ((fromJSON (readFile ./flake.lock)).nodes) nixpkgs gomod2nix;
    in
      import (fetchTree nixpkgs.locked) {
        overlays = [
          (import "${fetchTree gomod2nix.locked}/overlay.nix")
        ];
      }
  ),
}: let
  goEnv = pkgs.mkGoEnv {pwd = ./.;};
in
  pkgs.mkShell {
    packages = [
      pkgs.go
      pkgs.gopls
      pkgs.gotools
      pkgs.go-tools
      pkgs.delve
      goEnv
      pkgs.gomod2nix
    ];
  }
