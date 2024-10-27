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
      default = pkgs.callPackage ./default.nix {
        version = self.lastModifiedDate;
      };
    });
  };
}
