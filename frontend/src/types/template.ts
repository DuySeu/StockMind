export type Template = {
  id: string;
  name: string;
  description: string | null;
  prompt: string;
  rule_ids: string[];
  field: Record<string, string>;
  created_at: string;
  updated_at: string;
};
