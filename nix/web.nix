{
  lib,
  stdenvNoCC,
  buildGoModule,
  bun,
  nodejs,
  version,
  vendorHash,
}:

let
  inherit (lib) fileset;

  depsHash = (import ./web-deps-hashes.nix).${stdenvNoCC.hostPlatform.system} or null;

  manifest = fileset.unions [
    ../web/package.json
    ../web/bun.lock
  ];

  # Version-less name: the output path follows the name, so a version bump would fetch it again.
  deps = stdenvNoCC.mkDerivation {
    name = "ordo-web-deps";
    src = fileset.toSource {
      root = ../web;
      fileset = manifest;
    };
    nativeBuildInputs = [ bun ];
    dontConfigure = true;
    dontFixup = true;
    buildPhase = ''
      runHook preBuild
      export HOME="$TMPDIR"
      bun install --frozen-lockfile --ignore-scripts --no-progress
      runHook postBuild
    '';
    installPhase = ''
      runHook preInstall
      mkdir -p "$out"
      cp -R node_modules "$out/node_modules"
      runHook postInstall
    '';
    outputHashMode = "recursive";
    outputHashAlgo = "sha256";
    outputHash = depsHash;
  };

  wasm = buildGoModule {
    pname = "ordo-wasm";
    inherit version vendorHash;
    src = fileset.toSource {
      root = ../.;
      fileset = fileset.difference ../. (
        fileset.unions [
          ../native
          ../web
        ]
      );
    };
    buildPhase = ''
      runHook preBuild
      GOOS=js GOARCH=wasm CGO_ENABLED=0 go build -ldflags='-s -w' -o ordo.wasm ./cmd/ordo-wasm
      runHook postBuild
    '';
    doCheck = false;
    installPhase = ''
      runHook preInstall
      install -Dm644 ordo.wasm "$out/ordo.wasm"
      install -Dm644 "$(go env GOROOT)/lib/wasm/wasm_exec.js" "$out/wasm_exec.js"
      runHook postInstall
    '';
  };

  site =
    build:
    stdenvNoCC.mkDerivation {
      pname = "ordo-web-${build}";
      inherit version;
      src = fileset.toSource {
        root = ../web;
        fileset = fileset.unions [
          manifest
          ../web/svelte.config.js
          ../web/vite.config.ts
          ../web/tsconfig.json
          ../web/static
          (fileset.difference ../web/src (
            fileset.unions [
              (fileset.fileFilter (file: lib.hasInfix ".test." file.name) ../web/src)
              (fileset.maybeMissing ../web/src/lib/core/wasm)
            ]
          ))
        ];
      };
      nativeBuildInputs = [
        bun
        nodejs
      ];
      configurePhase = ''
        runHook preConfigure
        cp -R ${deps}/node_modules node_modules
        chmod -R u+w node_modules
        patchShebangs node_modules
        mkdir -p src/lib/core/wasm
        cp ${wasm}/ordo.wasm ${wasm}/wasm_exec.js src/lib/core/wasm/
        # tsconfig.json extends .svelte-kit's, which only a sync without ORDO_WEB writes.
        HOME="$TMPDIR" bun run prepare
        runHook postConfigure
      '';
      buildPhase = ''
        runHook preBuild
        export HOME="$TMPDIR"
        ORDO_WEB=${build} bun run build
        runHook postBuild
      '';
      installPhase = ''
        runHook preInstall
        cp -R build/${build} "$out"
        runHook postInstall
      '';
      passthru = { inherit deps wasm; };
    };
in
lib.optionalAttrs (depsHash != null) {
  web-site = site "site";
  web-private = site "private";
}
