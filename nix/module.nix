{ self }:
{
  config,
  lib,
  pkgs,
  ...
}:
let
  cfg = config.services.ordo-web;
  home = config.users.users.${cfg.user}.home;
in
{
  options.services.ordo-web = {
    enable = lib.mkEnableOption "`ordo serve`, which syncs the Ordo app and serves it to a browser";

    package = lib.mkOption {
      type = lib.types.package;
      default = self.packages.${pkgs.stdenv.hostPlatform.system}.ordo;
      defaultText = lib.literalExpression "ordo.packages.\${system}.ordo";
      description = "The ordo package to run.";
    };

    user = lib.mkOption {
      type = lib.types.str;
      example = "alice";
      description = ''
        Account the app runs as. It edits that user's projects in place, the
        same files the terminal app and Syncthing write, so it must be their
        owner.
      '';
    };

    group = lib.mkOption {
      type = lib.types.str;
      default = config.users.users.${cfg.user}.group;
      defaultText = lib.literalExpression "config.users.users.\${user}.group";
      description = ''
        Group the app runs as. With a unix socket, the proxy reaches it through
        this group, so give it one the proxy's user belongs to.
      '';
    };

    listen = lib.mkOption {
      type = lib.types.str;
      default = "127.0.0.1:7319";
      example = "unix:/run/ordo-web/web.sock";
      description = ''
        Address to listen on, `host:port` or `unix:/path`. The app speaks plain
        HTTP: keep it on loopback and put a TLS reverse proxy in front. A socket
        belongs in /run/ordo-web, the unit's runtime directory.
      '';
    };

    baseUrl = lib.mkOption {
      type = lib.types.str;
      example = "https://ordo.example.com";
      description = "Public address the proxy serves, printed in pairing links.";
    };

    dataDir = lib.mkOption {
      type = lib.types.str;
      default = "${home}/.local/share/ordo";
      defaultText = lib.literalExpression ''"''${home}/.local/share/ordo"'';
      description = ''
        Ordo's data directory: projects/ inside it, web.json (paired devices)
        beside them.
      '';
    };

    readWritePaths = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = [ ];
      example = [ "/home/alice/Sync/ordo" ];
      description = ''
        Extra paths the unit may write, for a projects directory that is a
        symlink into a synced folder.
      '';
    };
  };

  config = lib.mkIf cfg.enable {
    systemd.services.ordo-web = {
      description = "Ordo app server";
      wantedBy = [ "multi-user.target" ];
      after = [ "network.target" ];
      environment.ORDO_DATA_PATH = cfg.dataDir;
      serviceConfig = {
        ExecStart = lib.escapeShellArgs [
          (lib.getExe cfg.package)
          "serve"
          "--listen"
          cfg.listen
          "--base-url"
          cfg.baseUrl
        ];
        User = cfg.user;
        Group = cfg.group;
        RuntimeDirectory = "ordo-web";
        RuntimeDirectoryMode = "0750";
        Restart = "on-failure";
        RestartSec = "5s";

        NoNewPrivileges = true;
        PrivateTmp = true;
        PrivateDevices = true;
        ProtectSystem = "strict";
        ProtectHome = "read-only";
        ReadWritePaths = [ cfg.dataDir ] ++ cfg.readWritePaths;
        ProtectKernelTunables = true;
        ProtectKernelModules = true;
        ProtectKernelLogs = true;
        ProtectControlGroups = true;
        ProtectClock = true;
        ProtectHostname = true;
        ProtectProc = "invisible";
        ProcSubset = "pid";
        RestrictAddressFamilies = [
          "AF_UNIX"
          "AF_INET"
          "AF_INET6"
        ];
        RestrictNamespaces = true;
        RestrictRealtime = true;
        RestrictSUIDSGID = true;
        LockPersonality = true;
        CapabilityBoundingSet = [ "" ];
        SystemCallArchitectures = "native";
        SystemCallFilter = [
          "@system-service"
          "~@privileged"
          "~@resources"
        ];
      };
    };
  };
}
