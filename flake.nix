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

      mailpump-static = mailpump.overrideAttrs(old: {
        ldflags     = [ "-s" "-w" ];
        CGO_ENABLED = 0;
      });

      ci = pkgs.stdenvNoCC.mkDerivation rec {
        inherit (mailpump) version;

        pname = "mailpump-ci";

        dontUnpack = true;

        installPhase = ''
          mkdir -p $out

          cp "${mailpump-static}/bin/mailpump" \
            "$out/mailpump-${version}-${mailpump-static.stdenv.hostPlatform.system}"
          chmod 0755 "$out/mailpump-${version}-${mailpump-static.stdenv.hostPlatform.system}"

          cd "$out" && for i in *; do
            sha256sum -b "$i" > "$i.sha256"
          done
        '';
      };
    };

    devShell.x86_64-linux = self.packages.x86_64-linux.mailpump.overrideAttrs(old: {
      nativeBuildInputs = old.nativeBuildInputs ++ old.passthru.devTools;
    });
  };
}
