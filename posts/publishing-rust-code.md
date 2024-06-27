---
url: "publishing-rust-code"
title: "Publishing Rust code"
description: "Publishing rust packages to cargo and other package managers"
date: "2024-06-12"
---
# Publishing a Rust Package to Cargo and Other Package Managers

 I will be describing the process of publishing a Rust package to Cargo and other package managers. The code for a simple file traverser is provided as an example.

## The Code

```rust
use colored::Colorize;

use std::{fs, io, os::unix::fs::PermissionsExt};

fn menu(base_path: &str) -> io::Result<()> {
    let mut exit = false;

    while !exit {
        for entry in fs::read_dir(base_path)? {
            let entry = entry?;
            let file_path = entry.path();
            let file_metadata = fs::metadata(&file_path)?;

            let file_name = entry.file_name();
            if !file_metadata.is_file() {
                println!("{} ", file_name.to_string_lossy().blue(),);
            } else {
                println!(
                    "{} | Bytes: {} | Perms: {:?} ",
                    file_name.to_string_lossy(),
                    file_metadata.len(),
                    file_metadata.permissions().mode()
                );
            }
        }

        println!("Enter file name to navigate to");
        println!("Enter 'exit' to quit or press Enter to continue:");
        let mut input = String::new();
        io::stdin().read_line(&mut input)?;

        let input = input.trim(); // Remove leading and trailing whitespaces

        if input == "exit" {
            exit = true;
        } else {
            menu(&(base_path.to_owned() + input + "/"))?;
        }
    }

    Ok(())
}

fn main() {
    match menu("/") {
        Ok(()) => {
            println!("Menu executed successfully");
        }
        Err(err) => {
            eprintln!("Error: {}", err);
        }
    }
}

```

very simple, just a file-traverser ( like cd + ls but *crap*)


## Publishing to Cargo

Follow the steps in the Cargo reference: [https://doc.rust-lang.org/cargo/reference/publishing.html](https://doc.rust-lang.org/cargo/reference/publishing.html)

1. Log in using `cargo login` and add your API key.
2. Configure `Cargo.toml` with package metadata (name, version, authors, description, license, etc.).
3. Generate documentation using `cargo doc` and `cargo doc --open`. ( code below)
4. Perform a dry run with `cargo publish --dry-run`.
![Pasted image 20240612142219.png](https://img.kimbell.uk/NA0R6P.png)

5. Check the files included in the package using `cargo package --list`.
![Pasted image 20240612142326.png](https://img.kimbell.uk/Sc312w.png)
6. Publish the package using `cargo publish`.

```toml
[package]
name = "filetree"
version = "0.1.0"
edition = "2021"
authors = ["Seal <will@kimbell.uk>"]
description = "A Rust library for working with file trees"
license = "MIT"
repository = "https://github.com/seal/filetree-traversing"
homepage = "https://github.com/seal/filetree-traversing"
readme = "README.md"
keywords = ["file", "tree", "directory", "filesystem"]
categories = ["filesystem"]

[dependencies]
colored = "2.1.0"

[badges]
maintenance = { status = "actively-developed" }

[package.metadata.docs.rs]
all-features = true
rustdoc-args = ["--cfg", "docsrs"]

```

#### Cargo doc

Here's our updated code for using Cargo doc functionality.
```rust
//! A module for navigating and displaying file information in a directory.
//!
//! This module provides a `menu` function that allows the user to navigate through directories
//! and view information about files, such as file size and permissions.

use colored::Colorize;
use std::{fs, io, os::unix::fs::PermissionsExt};
/// Displays a menu for navigating and displaying file information in a directory.
///
/// The `menu` function reads the contents of the directory specified by `base_path` and displays
/// the file names, sizes, and permissions. It also allows the user to navigate to subdirectories
/// by entering the file name.
///
/// # Arguments
///
/// * `base_path` - The path to the directory to navigate and display file information for.
///
/// # Returns
///
/// Returns `Ok(())` if the menu executed successfully, or an `Err` if an I/O error occurred.
///
/// # Examples
///
/// ```
/// use filetree::menu;
///
/// match menu("/") {
///     Ok(()) => {
///         println!("Menu executed successfully");
///     }
///     Err(err) => {
///         eprintln!("Error: {}", err);
///     }
/// }
/// ```
pub fn menu(base_path: &str) -> io::Result<()> {
// Code is here, omitted for length 
}
/// The main entry point of the program.
///
/// This function calls the `menu` function with the root directory ("/") as the starting path.
/// It prints a success message if the menu executed successfully, or an error message if an error occurred.
fn main() {
    match menu("/") {
        Ok(()) => {
            println!("Menu executed successfully");
        }
        Err(err) => {
            eprintln!("Error: {}", err);
        }
    }
}

```



You can *remove* files from crates via: 

```toml
[package]
# ...
exclude = [
    "public/assets/*",
    "videos/*",
]

```

### Resolving Issues
![Pasted image 20240612142811.png](https://img.kimbell.uk/FZ94F1.png)


The error ? 
*filetree* crate already exists.
Updating steps:
- Update Cargo.toml version ( using https://semver.org/ versioning ) ( we'll +1 to patch in this case)
```toml
version = "0.1.1"
```
- run `cargo package`
- run `cargo publish`

Then the package is live:
https://crates.io/crates/filetree-traversing

So to recap / publish a new version we do:

1. Update Cargo.toml
2. cargo package 
3. cargo publish 
4. cargo yank --version $OLD_VERSION


## Publishing Binaries with GitHub CI

Use cargo-dist to publish binaries: [https://github.com/axodotdev/cargo-dist](https://github.com/axodotdev/cargo-dist)

1. Install cargo-dist: `cargo install cargo-dist`
2. Set up cargo-dist: `cargo dist init`
3. Commit and push changes.
4. Build with `cargo dist build`.

Example release steps:

```sh
git commit -am "release: version 0.1.6" 
git push
git tag "v0.1.6"
git push --tags
cargo package 
cargo publish
```


Here we can now see our CI is failing due to unix package not working on windows
![Pasted image 20240612150202.png](https://img.kimbell.uk/JZv3RJ.png)



We'll fix via:
```rust
use std::{fs, io};
file_metadata.permissions().readonly()
```


We publish again via our previous commands

```
// edit cargo.toml
git commit -am "release: version 0.1.7"
git push
git tag "v0.1.6"
git push --tags
cargo package
cargo publish
```


Now we can see our CI has completed and we get this link:
https://github.com/seal/filetree-traversing/releases/tag/v0.1.7

With our macOS, arm & x86 mac, and tar for linux builds done 

Now this is *great* but we have *no package mangaers*

cargo-dist supports shell, powershell, npm, homebrew, and msi.

1. Run `cargo dist init` to enable package managers.
2. Configure each package manager.

We'll enable shell, powershell & msi.

### AUR (Arch User Repository)

1. Create a new repo for the AUR package.
2. Add a `PKGBUILD` file with package details. ( code below)
3. Add a `LICENSE` file. ( I used MIT)
4. Generate `.SRCINFO` using `makepkg --printsrcinfo > .SRCINFO`.
5. Test the package with `makepkg -si`.
6. Delete all files except `.SRCINFO`, `PKGBUILD`, and `LICENSE`.
7. Push the package to the AUR repository.

Shell steps:
```
git init 
git add .
git commit -m "Whatever"
git remote add origin ssh://aur@aur.archlinux.org/filetree-traversing.git
git branch -M master // aur on master, sad times
git push origin master
```

PKGBUILD:
```
# Maintainer: Seal <will@kimbell.uk>

pkgname=filetree-traversing
pkgver=0.1.13
pkgrel=1
pkgdesc="A Rust library for working with file trees"
arch=('x86_64')
url="https://github.com/seal/filetree-traversing"
license=('MIT')
depends=('rust' 'cargo')
makedepends=('cargo')
source=("$pkgname-$pkgver.tar.gz::$url/archive/refs/tags/v$pkgver.tar.gz")
sha256sums=('SKIP')  # Replace with the actual SHA-256 checksum

build() {
    cd "$srcdir/$pkgname-$pkgver"
    cargo build --release
}

package() {
    cd "$srcdir/$pkgname-$pkgver"
    install -Dm755 "target/release/$pkgname" "$pkgdir/usr/bin/$pkgname"
    install -Dm644 LICENSE "$pkgdir/usr/share/licenses/$pkgname/LICENSE"
    install -Dm644 README.md "$pkgdir/usr/share/doc/$pkgname/README.md"
}

```


And then yes, you guessed it
https://aur.archlinux.org/packages/filetree-traversing
wohoo! 

Install via `yay -S filetree-traversing`


Now to update your package for new versions, follow theses steps:

First:
```
edit Cargo.toml
git add .
git commit -am "release: version 0.1.X"
git push
git tag "v0.1.X"
git push --tags
cargo package
cargo publish
cargo yank --version=OLD_VERSION
```

Second:
```
git clone ssh://aur@aur.archlinux.org/filetree-traversing.git
// Change PKGBUILD
makepkg --printsrcinfo > .SRCINFO
git add . && git commit -m "Update to v0.X.X" && git push 
```
