{
  description = "My collection of custom waybar widgets";

  inputs = { nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable"; };

  outputs = inputs@{ self, nixpkgs, ... }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f system);
    in
    {
      overlay = final: prev: import ./nix/pkgs/default.nix { pkgs = final; };

      hmModules.pomo = import ./nix/hm-modules/pomo.nix;
      hmModules.default = self.hmModules.pomo;

      packages = forAllSystems
        (system:
          let
            pkgs = import nixpkgs { inherit system; };
            all = import ./nix/pkgs/default.nix { inherit pkgs; };
          in
          all // { default = all.waybar-widgets; });
    };
}
