use std::io::Result;

fn main() -> Result<()> {
    tonic_prost_build::configure()
        .build_server(true)
        .build_client(false)
        .type_attribute(
            "astra.sync",
            "#[derive(serde::Serialize, serde::Deserialize)]",
        )
        .compile_protos(&["proto/sync.proto"], &["proto"])?;
    Ok(())
}
