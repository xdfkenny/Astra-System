//! Disk pressure injection for edge-device resource exhaustion testing.
//!
//! Simulates disk I/O contention and near-full filesystem conditions that
//! can cause SQLite database locking, write amplification, and storage
//! operation timeouts in the sync-daemon's encrypted database.
//!
//! # Safety
//!
//! This module writes files to a configurable target directory and fills
//! disk space up to the specified limit. It is **destructive** and should
//! only be run in isolated test environments (containers, VMs, or dedicated
//! test kiosks). The `--dry-run` flag prints the plan without touching disk.

use std::io::{self, Write};
use std::path::PathBuf;
use std::sync::atomic::{AtomicBool, AtomicU64, Ordering};
use std::sync::Arc;
use std::thread;
use std::time::{Duration, Instant};

use rand::Rng;
use tracing::{info, warn};

/// Configuration for disk pressure injection.
#[derive(Debug, Clone)]
pub struct DiskPressureConfig {
    /// Target directory to write pressure files into.
    pub target_dir: PathBuf,
    /// Total bytes to write across all pressure files (0 = 80% of available).
    pub target_bytes: u64,
    /// I/O operation type: "write" sequential, "randwrite", or "read" contention.
    pub io_pattern: String,
    /// Block size for each write (default 4 KiB).
    pub block_size: usize,
    /// Number of concurrent writer threads (default 4).
    pub concurrency: usize,
    /// Duration to sustain pressure before cleanup.
    pub duration: Duration,
    /// If true, print the plan without executing.
    pub dry_run: bool,
}

impl Default for DiskPressureConfig {
    fn default() -> Self {
        Self {
            target_dir: PathBuf::from("/tmp/astra-chaos-disk"),
            target_bytes: 0,
            io_pattern: "write".to_string(),
            block_size: 4096,
            concurrency: 4,
            duration: Duration::from_secs(30),
            dry_run: false,
        }
    }
}

/// Statistics collected during a disk pressure run.
#[derive(Debug, Default, Clone)]
pub struct DiskPressureStats {
    pub bytes_written: u64,
    pub io_errors: u64,
    pub files_created: u64,
    #[allow(dead_code)]
    pub duration_secs: f64,
    pub max_latency_us: u64,
    pub p99_latency_us: u64,
    pub avg_latency_us: u64,
}

/// Tracks I/O latency in a lock-free manner.
#[derive(Debug)]
struct LatencyTracker {
    count: AtomicU64,
    total_us: AtomicU64,
    max_us: AtomicU64,
    p99_us: AtomicU64,
}

impl LatencyTracker {
    fn new() -> Self {
        Self {
            count: AtomicU64::new(0),
            total_us: AtomicU64::new(0),
            max_us: AtomicU64::new(0),
            p99_us: AtomicU64::new(0),
        }
    }

    fn record(&self, latency: Duration) {
        let us = latency.as_micros() as u64;
        self.count.fetch_add(1, Ordering::Relaxed);
        self.total_us.fetch_add(us, Ordering::Relaxed);
        let prev = self.max_us.load(Ordering::Relaxed);
        if us > prev {
            self.max_us.store(us, Ordering::Relaxed);
        }
    }

    fn finalize(&self) -> (u64, u64, u64) {
        let count = self.count.load(Ordering::Relaxed);
        let max = self.max_us.load(Ordering::Relaxed);
        let p99 = self.p99_us.load(Ordering::Relaxed);
        let avg = if count > 0 { self.total_us.load(Ordering::Relaxed) / count } else { 0 };
        (max, p99, avg)
    }
}

/// Run disk pressure injection according to the given config.
pub fn run(config: &DiskPressureConfig) -> Result<DiskPressureStats, Box<dyn std::error::Error>> {
    if config.dry_run {
        info!("[dry-run] Disk pressure plan:");
        info!("  target_dir:     {:?}", config.target_dir);
        info!("  target_bytes:   {}", config.target_bytes);
        info!("  io_pattern:     {}", config.io_pattern);
        info!("  block_size:     {}", config.block_size);
        info!("  concurrency:    {}", config.concurrency);
        info!("  duration:       {:?}", config.duration);
        return Ok(DiskPressureStats::default());
    }

    let stop = Arc::new(AtomicBool::new(false));
    let stats = Arc::new(std::sync::Mutex::new(DiskPressureStats::default()));
    let latency = Arc::new(LatencyTracker::new());

    // Ensure target directory exists
    std::fs::create_dir_all(&config.target_dir)?;

    let available = match fs_available(&config.target_dir) {
        Some(bytes) => bytes,
        None => {
            warn!("could not determine available disk space, using target_bytes as-is");
            config.target_bytes
        }
    };

    let target = if config.target_bytes == 0 {
        // Default: fill to 80% of available space
        (available as f64 * 0.8) as u64
    } else {
        config.target_bytes.min(available.saturating_sub(256 * 1024 * 1024)) // leave 256 MiB
    };

    info!(
        available_bytes = available,
        target_bytes = target,
        "Disk pressure target computed"
    );

    if target == 0 {
        warn!("target is 0 bytes, skipping disk pressure");
        return Ok(DiskPressureStats::default());
    }

    let stop_signal = stop.clone();
    let stats_clone = stats.clone();
    let latency_clone = latency.clone();
    let dir = config.target_dir.clone();
    let block = config.block_size;
    let pattern = config.io_pattern.clone();
    let concurrency = config.concurrency;
    let chunk_target = target / concurrency as u64;

    let handles: Vec<_> = (0..concurrency)
        .map(|worker_id| {
            let stop = stop_signal.clone();
            let stats = stats_clone.clone();
            let lt = latency_clone.clone();
            let dir = dir.clone();
            let pattern = pattern.clone();
            let block = block;

            thread::spawn(move || {
                let file_path = dir.join(format!("pressure_{}.blob", worker_id));

                match pattern.as_str() {
                    "randwrite" => {
                        randwrite_worker(&file_path, block, chunk_target, &stop, &stats, &lt);
                    }
                    "read" => {
                        read_worker(&file_path, block, &stop, &stats);
                    }
                    _ => {
                        write_worker(&file_path, block, chunk_target, &stop, &stats, &lt);
                    }
                }
            })
        })
        .collect();

    thread::sleep(config.duration);
    stop.store(true, Ordering::SeqCst);

    for h in handles {
        h.join().map_err(|e| format!("thread join failed: {:?}", e))?;
    }

    // Cleanup
    cleanup(&config.target_dir)?;

    let (max_lat, p99_lat, avg_lat) = latency.finalize();
    let mut final_stats = stats.lock().unwrap_or_else(|e| e.into_inner()).clone();
    final_stats.max_latency_us = max_lat;
    final_stats.p99_latency_us = p99_lat;
    final_stats.avg_latency_us = avg_lat;

    info!(
        bytes_written = final_stats.bytes_written,
        io_errors = final_stats.io_errors,
        files_created = final_stats.files_created,
        max_latency_us = max_lat,
        p99_latency_us = p99_lat,
        avg_latency_us = avg_lat,
        "Disk pressure completed"
    );

    Ok(final_stats)
}

fn write_worker(
    path: &PathBuf,
    block_size: usize,
    max_bytes: u64,
    stop: &AtomicBool,
    stats: &Arc<std::sync::Mutex<DiskPressureStats>>,
    latency: &LatencyTracker,
) {
    let mut buf = vec![0u8; block_size];
    let mut rng = rand::thread_rng();
    rng.fill(&mut buf[..]);

    let mut written: u64 = 0;
    let mut file = match std::fs::File::create(path) {
        Ok(f) => f,
        Err(e) => {
            warn!(%e, "failed to create pressure file");
            return;
        }
    };

    while !stop.load(Ordering::SeqCst) && written < max_bytes {
        let start = Instant::now();
        match file.write_all(&buf) {
            Ok(()) => {
                latency.record(start.elapsed());
                written += block_size as u64;
                if let Ok(mut s) = stats.lock() {
                    s.bytes_written += block_size as u64;
                }
            }
            Err(e) if e.kind() == io::ErrorKind::StorageFull => {
                warn!("disk full, stopping write worker");
                if let Ok(mut s) = stats.lock() {
                    s.io_errors += 1;
                }
                break;
            }
            Err(e) => {
                warn!(%e, "write error");
                if let Ok(mut s) = stats.lock() {
                    s.io_errors += 1;
                }
            }
        }

        // fsync periodically to increase pressure on SQLite
        if written % (block_size as u64 * 256) == 0 {
            let start = Instant::now();
            let _ = file.sync_all();
            latency.record(start.elapsed());
        }
    }

    if let Ok(mut s) = stats.lock() {
        s.files_created += 1;
    }
}

fn randwrite_worker(
    path: &PathBuf,
    block_size: usize,
    max_bytes: u64,
    stop: &AtomicBool,
    stats: &Arc<std::sync::Mutex<DiskPressureStats>>,
    latency: &LatencyTracker,
) {
    let file = match std::fs::File::create(path) {
        Ok(f) => f,
        Err(e) => {
            warn!(%e, "failed to create pressure file");
            return;
        }
    };

    // Pre-allocate
    file.set_len(max_bytes).ok();

    let mut rng = rand::thread_rng();
    let buf = vec![0u8; block_size];
    let mut written: u64 = 0;

    while !stop.load(Ordering::SeqCst) && written < max_bytes / 4 {
        let offset = rng.gen_range(0..max_bytes.saturating_sub(block_size as u64));
        #[cfg(unix)]
        {
            use std::os::unix::fs::FileExt;
            let start = Instant::now();
            match file.write_at(&buf, offset) {
                Ok(n) => {
                    latency.record(start.elapsed());
                    written += n as u64;
                    if let Ok(mut s) = stats.lock() {
                        s.bytes_written += n as u64;
                    }
                }
                Err(e) => {
                    warn!(%e, "randwrite error at offset {}", offset);
                    if let Ok(mut s) = stats.lock() {
                        s.io_errors += 1;
                    }
                }
            }
        }
        #[cfg(not(unix))]
        {
            let _ = offset;
            thread::sleep(std::time::Duration::from_millis(1));
        }
    }
}

fn read_worker(
    path: &PathBuf,
    block_size: usize,
    stop: &AtomicBool,
    stats: &Arc<std::sync::Mutex<DiskPressureStats>>,
) {
    // Read contention: open existing file and read random offsets
    let file = match std::fs::File::open(path) {
        Ok(f) => f,
        Err(_) => return, // no file to read
    };

    let file_len = match file.metadata().map(|m| m.len()) {
        Ok(len) => len,
        Err(_) => return,
    };

    let mut rng = rand::thread_rng();
    let mut buf = vec![0u8; block_size];

    #[cfg(unix)]
    {
        use std::os::unix::fs::FileExt;
        while !stop.load(Ordering::SeqCst) {
            let offset = rng.gen_range(0..file_len.saturating_sub(block_size as u64));
            match file.read_at(&mut buf, offset) {
                Ok(_) => {
                    if let Ok(mut s) = stats.lock() {
                        s.bytes_written += block_size as u64;
                    }
                }
                Err(e) => {
                    warn!(%e, "read contention error");
                    if let Ok(mut s) = stats.lock() {
                        s.io_errors += 1;
                    }
                }
            }
            thread::sleep(Duration::from_millis(10));
        }
    }
    #[cfg(not(unix))]
    {
        let _ = stop;
    }
}

/// Remove all pressure files from the target directory.
fn cleanup(dir: &PathBuf) -> Result<(), Box<dyn std::error::Error>> {
    if dir.exists() {
        for entry in std::fs::read_dir(dir)? {
            let entry = entry?;
            let path = entry.path();
            if path
                .file_name()
                .and_then(|n| n.to_str())
                .map(|n| n.starts_with("pressure_") && n.ends_with(".blob"))
                .unwrap_or(false)
            {
                std::fs::remove_file(&path)?;
                info!(path = %path.display(), "cleaned up pressure file");
            }
        }
    }
    Ok(())
}

/// Get available bytes on the filesystem containing `path`.
fn fs_available(path: &PathBuf) -> Option<u64> {
    #[cfg(unix)]
    {
        let _meta = std::fs::metadata(path).ok()?;
        nix::sys::statvfs::statvfs(path)
            .ok()
            .map(|s| s.blocks_available() as u64 * s.block_size() as u64)
    }
    #[cfg(not(unix))]
    {
        let _ = path;
        None
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::tempdir;

    #[test]
    fn disk_pressure_dry_run_returns_immediately() {
        let config = DiskPressureConfig {
            dry_run: true,
            ..Default::default()
        };
        let stats = run(&config).unwrap();
        assert_eq!(stats.bytes_written, 0);
    }

    #[test]
    fn cleanup_removes_pressure_files() {
        let dir = tempdir().unwrap();
        let pressure_file = dir.path().join("pressure_0.blob");
        std::fs::write(&pressure_file, b"test").unwrap();

        // Create a non-pressure file that should NOT be removed
        let other = dir.path().join("keep_me.txt");
        std::fs::write(&other, b"keep").unwrap();

        cleanup(&dir.path().to_path_buf()).unwrap();
        assert!(!pressure_file.exists());
        assert!(other.exists());
    }
}
