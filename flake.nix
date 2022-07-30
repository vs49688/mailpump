{
  description = "MailPump";

  outputs = { self, nixpkgs }: {
    overlays = {
      default = final: prev: {
        mailpump = prev.callPackage ./default.nix {
          version = self.lastModifiedDate;

          buildGoModule = prev.buildGo118Module;
          gotools = prev.gotools.override {
            buildGoModule = prev.buildGo118Module;
          };
          gosec = prev.gosec.override {
            buildGoModule = prev.buildGo118Module;
          };

          mockgen = prev.mockgen.override { buildGoModule = prev.buildGo118Module; };
        };
      };
    };

    packages.x86_64-linux = let
      pkgs = import nixpkgs {
        system = "x86_64-linux";
        overlays = [ self.overlays.default ];
      };
    in rec {
      inherit (pkgs) mailpump;

      default = mailpump;
    };

    devShell.x86_64-linux = self.packages.x86_64-linux.mailpump.overrideAttrs(old: {
      nativeBuildInputs = old.nativeBuildInputs ++ old.passthru.devTools;
    });
  };
}
