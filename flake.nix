{
  description = "Blast with cardano hashes generation ";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    gomod2nix.url = "github:nix-community/gomod2nix";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
    gomod2nix,
  }:
    flake-utils.lib.eachDefaultSystem (system: let
      pkgs = import nixpkgs {
        inherit system;
        overlays = [gomod2nix.overlays.default];
      };
    in {
      packages.default = (pkgs.callPackage ./. {}).overrideAttrs (_: {doCheck = false;});
      devShells.default = import ./shell.nix {inherit pkgs;};
    });
}
