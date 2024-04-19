{ buildGoModule
, version
}:

buildGoModule {
  inherit version;

  pname = "mailpump";

  src = ./.;

  vendorHash = null;
}