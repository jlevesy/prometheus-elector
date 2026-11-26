{
  description = "Development environment for prometheus-elector";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs = {
    self,
    nixpkgs,
  }: let
    systems = ["x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin"];
    forAllSystems = nixpkgs.lib.genAttrs systems;
  in {
    devShells = forAllSystems (system: let
      pkgs = nixpkgs.legacyPackages.${system};
    in {
      default = pkgs.mkShell {
        packages = [
          pkgs.go_1_26
          pkgs.golangci-lint
          pkgs.gnumake
          pkgs.gcc
          pkgs.gh
          pkgs.ko
          pkgs.kubernetes-helm
          pkgs.kubectl
          pkgs.k3d
        ];

        env.GOTOOLCHAIN = "local";
      };
    });
  };
}
