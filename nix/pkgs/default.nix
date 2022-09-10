{ pkgs }: rec {
  waybar-widgets = pkgs.callPackage ./waybar-widgets.nix {};
}
