{ pkgs, lib, config, ... }:

with lib;

let

  cfg = config.services.pomo;

  environmentFile = pkgs.writeText "pomo.env"
    (concatStringsSep "\n" (mapAttrsToList (k: v: "${k}=${v}") cfg.settings));

  pomo = "${cfg.package}/bin/waybar-widgets pomo";

  pomoBin = pkgs.writers.writeDashBin "pomo" ''
    ${pomo} "$@"
  '';

in
{
  options.services.pomo = {
    enable = mkEnableOption "pomo";

    package = mkOption {
      type = types.package;
      default = pkgs.callPackage ../pkgs/waybar-widgets.nix { };
    };

    idleTimeout = mkOption {
      type = types.ints.unsigned;
      default = 30;
      description = "Swayidle timeout in seconds";
    };

    settings = mkOption {
      default = { };
      type = types.attrsOf types.str;
    };
  };

  config = mkIf cfg.enable {
    services.pomo.settings.POMO_IDLE_TIMEOUT = "${cfg.idleTimeout}s";

    systemd.user.services.pomo = {
      Unit = {
        Description = "Pomodoro timer";
        PartOf = "graphical-session.target";
        After = [ "pomo.socket" ];
        Requires = [ "pomo.socket" ];
      };
      Service = {
        Type = "simple";
        ExecStart = "${pomo} server";
        EnvironmentFile = [ environmentFile ];
      };
    };

    systemd.user.sockets.pomo = {
      Unit.Description = "Socket for pomodoro timer";
      Socket.ListenStream = "%t/waybar-widgets/pomo.sock";
      Install.WantedBy = [ "sockets.target" ];
    };

    services.swayidle.timeouts =
      [
        {
          timeout = cfg.idleTimeout;
          command = "${pomo} idle_start";
          resumeCommand = "${pomo} idle_stop";
        }
      ];
  };
}
