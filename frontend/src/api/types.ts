// Thin aliases over the types generated from api/openapi.yaml (schema.d.ts).
// Regenerate with `npm run gen:api`; never edit schema.d.ts by hand.
import type { components } from './schema';

type Schemas = components['schemas'];

export type OperationName = Schemas['OperationName'];
export type Operation = Schemas['Operation'];
export type CalculateRequest = Schemas['CalculateRequest'];
export type CalculateResponse = Schemas['CalculateResponse'];
export type OperationsResponse = Schemas['OperationsResponse'];
export type ServerErrorCode = Schemas['ErrorCode'];

/** Failures that happen before a well-formed server error is received. */
export type ClientErrorCode = 'NETWORK_ERROR' | 'TIMEOUT' | 'BAD_RESPONSE';

export type ErrorCode = ServerErrorCode | ClientErrorCode;
