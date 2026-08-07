import { cp, mkdir, rm, stat } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const scriptDirectory = path.dirname(fileURLToPath(import.meta.url));
const frontendRoot = path.resolve(scriptDirectory, "..");
const repositoryRoot = path.resolve(frontendRoot, "..");
const source = path.join(repositoryRoot, "web", "static", "assets");
const publicRoot = path.join(frontendRoot, "public");
const destination = path.join(publicRoot, "assets");

const sourceStats = await stat(source);
if (!sourceStats.isDirectory()) {
  throw new Error(`Asset source is not a directory: ${source}`);
}

const relativeDestination = path.relative(publicRoot, destination);
if (
  relativeDestination.startsWith("..") ||
  path.isAbsolute(relativeDestination)
) {
  throw new Error(`Refusing to synchronize outside public/: ${destination}`);
}

await mkdir(publicRoot, { recursive: true });
await rm(destination, { recursive: true, force: true });
await cp(source, destination, { recursive: true });

console.log(`Static assets synchronized from ${source} to ${destination}`);
