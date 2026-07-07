use std::env;
use std::path::PathBuf;

fn main() {
    let header = "include/verifone_pos.h";

    // Re-run this build script if the C header changes.
    println!("cargo:rerun-if-changed={header}");

    let bindings = bindgen::Builder::default()
        .header(header)
        .parse_callbacks(Box::new(bindgen::CargoCallbacks::new()))
        // Emit constants and a type alias for VxStatus so the safe wrapper can
        // treat status codes as plain integers.
        .default_enum_style(bindgen::EnumVariation::Consts)
        // Keep preprocessor constants signed so they match the C `int` return
        // type used throughout the SDK.
        .default_macro_constant_type(bindgen::MacroTypeVariation::Signed)
        .generate()
        .expect("Unable to generate Verifone POS bindings");

    let out_path = PathBuf::from(env::var("OUT_DIR").unwrap());
    bindings
        .write_to_file(out_path.join("bindings.rs"))
        .expect("Couldn't write Verifone POS bindings");
}
