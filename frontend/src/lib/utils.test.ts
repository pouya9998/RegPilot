import { describe, expect, it } from "vitest";
import { formatBytes, shortDigest } from "./utils";

describe("utils", () => {
  it("formats bytes", () => {
    expect(formatBytes(0)).toBe("—");
    expect(formatBytes(512)).toBe("512 B");
    expect(formatBytes(1024 * 420)).toContain("KB");
  });

  it("shortens digests", () => {
    expect(shortDigest("sha256:abcdefghijklmnopqrstuvwxyz")).toContain("…");
  });
});
