/* tslint:disable */
/* eslint-disable */

/**
 * Compile Lini source to SVG.
 *
 * Throws on any error-level diagnostic, with the compiler's own LSP-shaped
 * message (`play.lini:3:5: error: …`) as the thrown value.
 */
export function compile(src: string, assets: any): string;

/**
 * Compile to a full HTML page rather than a bare SVG — the `--format html`
 * output, for a preview pane that wants a self-contained document.
 */
export function compile_html(src: string, assets: any): string;

/**
 * Compile with `var()` references inlined and text outlined to paths — the
 * `--static` output. Self-contained for download, or for a canvas rasteriser.
 */
export function compile_static(src: string, assets: any): string;

/**
 * The source with every bit of sugar lowered to primitives — what `lini
 * desugar` prints. The teaching view.
 */
export function desugar(src: string): string;

/**
 * Every diagnostic as the JSON document `--json` emits — stable codes, spans,
 * severities, and machine-applicable fixes. Never throws: a file that cannot
 * compile still reports why, which is what an editor's gutter wants.
 */
export function diagnostics(src: string, assets: any): string;

/**
 * Canonical formatting — what `lini fmt` writes.
 */
export function format(src: string): string;

/**
 * The source as `<span class="lini-tok-…">` HTML — what `lini highlight`
 * prints, from the same scanner the VS Code and Zed grammars take their words
 * from [SPEC 22]. Never throws: highlighting is lexical, so a file
 * mid-keystroke still colours, which is exactly what a live editor wants.
 */
export function highlight(src: string): string;

/**
 * The stylesheet [`highlight`]'s markup wears — what `lini highlight --css`
 * prints [SPEC 18/20].
 *
 * A host with no `lini` binary beside it — a bundler plugin, an Astro or
 * Docusaurus integration — has no other way to reach the palette, and a copy
 * kept in its own tree goes monochrome the day the scanner names a class this
 * sheet does not paint. Shipping it from the module ties the two to one
 * engine, exactly as [`highlight`] and the grammars already are.
 */
export function highlight_css(): string;

/**
 * The compiler's version, so a page can show which engine it is running.
 */
export function version(): string;

export type InitInput = RequestInfo | URL | Response | BufferSource | WebAssembly.Module;

export interface InitOutput {
    readonly memory: WebAssembly.Memory;
    readonly compile: (a: number, b: number, c: number, d: number) => void;
    readonly compile_html: (a: number, b: number, c: number, d: number) => void;
    readonly compile_static: (a: number, b: number, c: number, d: number) => void;
    readonly desugar: (a: number, b: number, c: number) => void;
    readonly diagnostics: (a: number, b: number, c: number, d: number) => void;
    readonly format: (a: number, b: number, c: number) => void;
    readonly highlight: (a: number, b: number, c: number) => void;
    readonly highlight_css: (a: number) => void;
    readonly version: (a: number) => void;
    readonly __wbindgen_export: (a: number, b: number) => number;
    readonly __wbindgen_export2: (a: number, b: number, c: number, d: number) => number;
    readonly __wbindgen_add_to_stack_pointer: (a: number) => number;
    readonly __wbindgen_export3: (a: number, b: number, c: number) => void;
}

export type SyncInitInput = BufferSource | WebAssembly.Module;

/**
 * Instantiates the given `module`, which can either be bytes or
 * a precompiled `WebAssembly.Module`.
 *
 * @param {{ module: SyncInitInput }} module - Passing `SyncInitInput` directly is deprecated.
 *
 * @returns {InitOutput}
 */
export function initSync(module: { module: SyncInitInput } | SyncInitInput): InitOutput;

/**
 * If `module_or_path` is {RequestInfo} or {URL}, makes a request and
 * for everything else, calls `WebAssembly.instantiate` directly.
 *
 * @param {{ module_or_path: InitInput | Promise<InitInput> }} module_or_path - Passing `InitInput` directly is deprecated.
 *
 * @returns {Promise<InitOutput>}
 */
export default function __wbg_init (module_or_path?: { module_or_path: InitInput | Promise<InitInput> } | InitInput | Promise<InitInput>): Promise<InitOutput>;
