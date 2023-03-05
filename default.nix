{ buildGoModule, gotools, gosec, mockgen, govulncheck, version }:
buildGoModule {
  inherit version;

  pname = "mailpump";

  src = ./.;

  vendorSha256 = null;

  passthru.devTools = [
    gotools
    gosec
    mockgen
    govulncheck
  ];
}
