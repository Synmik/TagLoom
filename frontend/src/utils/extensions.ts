// One place for "what kind of file is this?".  Until now the panel, the modal
// and the gallery each kept their own copy of these lists and their own way of
// reading an extension, and they drifted: PDFs rendered in the modal while
// double-click still launched the system reader.
//
// The backend list is `SupportedExtensions` in `utils/fileutil.go`.  The image
// and video sets here are that list minus what a browser cannot display
// (TIFF, JPEG XL), which is why they look similar but are not identical.

const imageExtensions = new Set([
  ".jpg",
  ".jpeg",
  ".png",
  ".gif",
  ".webp",
  ".bmp",
  ".svg",
  ".avif",
]);

const videoExtensions = new Set([
  ".mp4",
  ".mov",
  ".avi",
  ".webm",
  ".mkv",
  ".wmv",
  ".flv",
  ".m4v",
  ".3gp",
  ".3g2",
  ".vob",
  ".ogv",
  ".mpg",
  ".mpeg",
  ".m2v",
  ".ts",
  ".mts",
  ".m2ts",
  ".asf",
  ".rm",
  ".amv",
  ".f4v",
  ".dv",
  ".mxf",
]);

// Markdown and plain text: no picture of their own, so the panel shows the
// file's text and the gallery shows a bundled format icon.
const textExtensions = new Set([".md", ".txt"]);

// What the fullscreen modal shows.  SVG stays out: the right panel draws it,
// but double-click hands it to the system viewer.  Videos are out on purpose
// too — double-clicking one opens the external player.  Markdown and text are
// in: their text is read through a binding and shown as text.
const modalPreviewExtensions = new Set([
  ".jpg",
  ".jpeg",
  ".png",
  ".gif",
  ".webp",
  ".bmp",
  ".avif",
  ".pdf",
  ".md",
  ".txt",
]);

/** Lowercased extension with the leading dot, e.g. `.pdf`. */
export const extensionOf = (path: string): string =>
  `.${path.split(".").pop()?.toLowerCase() ?? ""}`;

/** Badge text: uppercase extension, with JPG and JPEG both shown as JPEG. */
export const formatLabel = (path: string): string => {
  const ext = extensionOf(path).slice(1).toUpperCase();
  return ext === "JPG" || ext === "JPEG" ? "JPEG" : ext;
};

/** Rendered by an `<img>` in the panel or the modal. */
export const isImageFile = (path: string): boolean => imageExtensions.has(extensionOf(path));

/** Played by a `<video>` element. */
export const isVideoFile = (path: string): boolean => videoExtensions.has(extensionOf(path));

export const isPdfFile = (path: string): boolean => extensionOf(path) === ".pdf";

/** Read inline as text — the backend decodes it, the view shows a `<pre>`. */
export const isTextFile = (path: string): boolean => textExtensions.has(extensionOf(path));

/** True when a double-click opens the file in the preview modal instead of the OS viewer. */
export const isBrowserPreviewable = (path: string): boolean =>
  modalPreviewExtensions.has(extensionOf(path));
