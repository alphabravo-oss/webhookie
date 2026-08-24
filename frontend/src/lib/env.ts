declare const __APP_VERSION__: string | undefined;
export const APP_VERSION = typeof __APP_VERSION__ === 'undefined' ? '0.1.0-dev' : __APP_VERSION__;
export const API_BASE = '/api/v1';
export const IS_DEV = import.meta.env.DEV;
