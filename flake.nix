{
  description = "MailPump";

  inputs.nixpkgs.url = github:NixOS/nixpkgs;

  outputs = { self, nixpkgs }: let
    forAllSystems = function:
      nixpkgs.lib.genAttrs [
        "x86_64-linux"
        "aarch64-linux"
      ] (system: function nixpkgs.legacyPackages.${system});
  in {
    overlays = {
      default = final: prev: {
        mailpump = prev.callPackage ./default.nix {
          version = self.lastModifiedDate;
        };
      };
    };

    packages = forAllSystems (pkgs: rec {
      mailpump = pkgs.callPackage ./default.nix {
        version = self.lastModifiedDate;
      };

      mailpump-static = mailpump.overrideAttrs(old: {
        ldflags     = [ "-s" "-w" ];
        CGO_ENABLED = 0;
      });

      default = mailpump;

      ci = pkgs.stdenvNoCC.mkDerivation (finalAttrs: {
        inherit (mailpump) version;

        pname = "mailpump-ci";

        dontUnpack = true;

        installPhase = ''
          mkdir -p $out

          cp "${mailpump-static}/bin/mailpump" \
            "$out/mailpump-${finalAttrs.version}-${mailpump-static.stdenv.hostPlatform.system}"
          chmod 0755 "$out/mailpump-${finalAttrs.version}-${mailpump-static.stdenv.hostPlatform.system}"

          cd "$out" && for i in *; do
            sha256sum -b "$i" > "$i.sha256"
          done
        '';
      });
    });
  };
}
