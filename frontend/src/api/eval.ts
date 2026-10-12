import api from "./index";
import { postSSE, type ChatEvent } from "./chat";

// Modules the eval page can run alone, each one a POST /v1/eval/<module>
export type EvalModule = "guardrail" | "sanitize" | "plan" | "pipeline" | "agent" | "tool";

export interface EvalAgentInfo {
  name: string;
  description: string;
  tools: string[] | null;
}

export interface EvalToolInfo {
  name: string;
  description: string;
  schema: Record<string, unknown>;
}

export interface EvalCatalog {
  agents: EvalAgentInfo[];
  tools: EvalToolInfo[];
  guardrailEnabled: boolean;
}

// GET /v1/eval/catalog - Load the agents and tools the forms pick from
export const getEvalCatalog = async (): Promise<EvalCatalog> => {
  const { data } = await api.get<EvalCatalog>("/eval/catalog");
  return data;
};

// POST /v1/eval/<module> - Run one module and stream its trace and result
export const runEvalModule = (
  module: EvalModule,
  input: Record<string, unknown>,
  onEvent: (event: ChatEvent) => void,
  onError: (error: unknown) => void,
  signal?: AbortSignal,
): Promise<void> =>
  postSSE(`/eval/${module}`, { "Content-Type": "application/json" }, JSON.stringify(input), onEvent, onError, signal);
