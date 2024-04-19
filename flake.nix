{
  description = "MailPump";

  inputs.nixpkgs.url = github:NixOS/nixpkgs;

  outputs = { self, nixpkgs }: {
    overlays = {
      default = final: prev: {
        mailpump = prev.callPackage ./default.nix {
          version = self.lastModifiedDate;
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

    devShells.x86_64-linux.default = self.packages.x86_64-linux.mailpump.overrideAttrs(old: {
      nativeBuildInputs = old.nativeBuildInputs ++ old.passthru.devTools;
    });
  };
}
