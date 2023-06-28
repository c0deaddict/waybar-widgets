{ lib, buildGoModule, makeWrapper, libnotify, zfs }:

buildGoModule rec {
  pname = "waybar-widgets";
  version = "0.0.1";

  src = ../..;

  vendorSha256 = "sha256-37PbRnArjdhq2bRCJFJa4xDyuPV8uRvtA9wW6zXTgKc=";

  subPackages = [ "cmd/waybar-widgets" ];

  nativeBuildInputs = [ makeWrapper ];

  postInstall = ''
    wrapProgram $out/bin/waybar-widgets \
      --prefix PATH : ${lib.makeBinPath [libnotify zfs]}
  '';

  meta = with lib; {
    description = "My collection of custom waybar widgets";
    homepage = "https://github.com/c0deaddict/waybar-widgets";
    license = licenses.mit;
    maintainers = with maintainers; [ c0deaddict ];
  };
}
