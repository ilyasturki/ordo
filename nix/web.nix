{
  lib,
  stdenv,
  bun,
  nodejs,
  version,
}:

let
  depsHash = (import ./web-deps-hashes.nix).${stdenv.hostPlatform.system} or null;

  manifest = lib.fileset.unions [
    ../app/package.json
    ../app/bun.lock
  ];

  # Version-less name: the output path follows the name, so a version bump would fetch it again.
  deps = stdenv.mkDerivation {
    name = "ordo-web-deps";

    src = lib.fileset.toSource {
      root = ../app;
      fileset = manifest;
    };

    nativeBuildInputs = [ bun ];

    dontConfigure = true;
    dontFixup = true;

    buildPhase = ''
      runHook preBuild
      export HOME="$TMPDIR"
      bun install --frozen-lockfile --ignore-scripts
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
in
if depsHash == null then
  null
else
  stdenv.mkDerivation {
    pname = "ordo-web";
    inherit version;

    src = lib.fileset.toSource {
      root = ../app;
      fileset = lib.fileset.unions [
        manifest
        ../app/tsconfig.json
        ../app/vite.config.ts
        (lib.fileset.difference ../app/src (
          lib.fileset.fileFilter (file: lib.hasInfix ".test." file.name) ../app/src
        ))
        (lib.fileset.difference ../app/static (
          lib.fileset.unions [
            (lib.fileset.maybeMissing ../app/static/ordo-core.wasm)
            (lib.fileset.maybeMissing ../app/static/wasm_exec.js)
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
      runHook postConfigure
    '';

    buildPhase = ''
      runHook preBuild
      export HOME="$TMPDIR"
      bun run build
      runHook postBuild
    '';

    installPhase = ''
      runHook preInstall
      cp -R build "$out"
      runHook postInstall
    '';

    passthru = { inherit deps; };
  }
