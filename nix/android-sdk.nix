# The Android SDK the native app builds with: Compose needs platform 37, AGP 9.5 looks for
# build-tools 36.0.0 and can't install it into the store, and gomobile needs the NDK.
{ nixpkgs, system }:
let
  pkgs = import nixpkgs {
    inherit system;
    config = {
      allowUnfree = true;
      android_sdk.accept_license = true;
    };
  };
in
(pkgs.androidenv.composeAndroidPackages {
  platformVersions = [ "37.0" ];
  buildToolsVersions = [ "36.0.0" ];
  includeNDK = true;
  ndkVersions = [ "29.0.14206865" ];
  includeCmake = false;
}).androidsdk
