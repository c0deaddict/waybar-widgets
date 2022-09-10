{ lib, buildGoModule, makeWrapper, libnotify }:

buildGoModule rec {
  pname = "waybar-widgets";
  version = "0.0.1";

  src = ../..;

  vendorSha256 = "sha256-0hHEallm8Z3yN3Zf0cCWrxPi5RsBHpySfW7t4id858c=";

  subPackages = [ "cmd/waybar-widgets" ];

  nativeBuildInputs = [ makeWrapper ];

  postInstall = ''
    wrapProgram $out/bin/waybar-widgets \
      --prefix PATH : ${lib.makeBinPath [libnotify]}
  '';

  meta = with lib; {
    description = "My collection of custom waybar widgets";
    homepage = "https://github.com/c0deaddict/waybar-widgets";
    license = licenses.mit;
    maintainers = with maintainers; [ c0deaddict ];
  };
}
