import DiskArbitration
import Foundation

class DiskMonitor {
    private var session: DASession?
    private let logFile: FileHandle?
    private let logPath = "\(NSHomeDirectory())/.gomediaimport-launchagent.log"

    init() {
        // Ensure the directory exists
        let logDirectory = (logPath as NSString).deletingLastPathComponent
        do {
            try FileManager.default.createDirectory(
                atPath: logDirectory, withIntermediateDirectories: true)
        } catch {
            print("Error creating log directory: \(error)")
            logFile = nil
            return
        }

        // Open or create the log file
        if FileManager.default.fileExists(atPath: logPath) {
            logFile = FileHandle(forUpdatingAtPath: logPath)
            logFile?.seekToEndOfFile()
        } else {
            FileManager.default.createFile(atPath: logPath, contents: nil)
            logFile = FileHandle(forWritingAtPath: logPath)
        }

        if logFile == nil {
            print("Failed to open log file")
        }
    }

    deinit {
        logFile?.closeFile()
    }

    private func log(_ message: String) {
        let timestamp = ISO8601DateFormatter().string(from: Date())
        let logMessage = "[\(timestamp)] \(message)\n"
        logFile?.write(logMessage.data(using: .utf8)!)
        print(logMessage)  // Also print to console for debugging
    }

    func start() {
        do {
            try runMonitor()
        } catch {
            log("Error starting monitor: \(error)")
        }
    }

    private func runMonitor() throws {
        // Create a session
        session = DASessionCreate(kCFAllocatorDefault)

        guard let session = session else {
            throw NSError(
                domain: "DiskMonitor", code: 1,
                userInfo: [
                    NSLocalizedDescriptionKey: "Failed to create DASession"
                ])
        }

        // Register for disk description changed callback
        DARegisterDiskDescriptionChangedCallback(
            session,
            nil,  // Match all disks
            nil,  // Watch all keys
            {
                (disk: DADisk, keys: CFArray, context: UnsafeMutableRawPointer?)
                in
                let monitor = Unmanaged<DiskMonitor>.fromOpaque(context!)
                    .takeUnretainedValue()
                monitor.handleDiskChanged(disk: disk)
            },
            Unmanaged.passUnretained(self).toOpaque()
        )

        // Schedule the session on the main dispatch queue
        DASessionScheduleWithRunLoop(
            session, CFRunLoopGetMain(), CFRunLoopMode.defaultMode.rawValue)

        log("Disk monitor started. Waiting for mount events...")
    }

    private func handleDiskChanged(disk: DADisk) {
        guard let description = DADiskCopyDescription(disk) as? [String: Any]
        else {
            log("Failed to get disk description")
            return
        }

        // Check if this is a mount event by looking for the DAVolumePath key
        if let volumePath = description["DAVolumePath"] as? URL {
            log("Volume mounted:")

            // Log relevant information about the mounted volume
            let keysToShow = [
                "DAVolumePath", "DAVolumeName", "DAVolumeKind", "DAMediaName",
                "DAMediaSize",
            ]

            for key in keysToShow {
                if let value = description[key] {
                    log("\(key): \(value)")
                }
            }

            // Launch gomediaimport with the volume path
            let path = volumePath.path
            launchGomediaimport(with: path)

            log("------------------------")
        }
    }

    private func launchGomediaimport(with path: String) {
        let task = Process()
        task.executableURL = URL(fileURLWithPath: "\(NSHomeDirectory())/go/bin/gomediaimport")
        task.arguments = [path]

        do {
            try task.run()
            log("Launched gomediaimport with path: \(path)")
        } catch {
            log("Failed to launch gomediaimport: \(error)")
        }
    }
}

// Create and start the disk monitor
let monitor = DiskMonitor()
monitor.start()

// Run the main run loop
RunLoop.main.run()
