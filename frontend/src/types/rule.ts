export type Rule = {
  id: string;
  name: string;
  description: string;
  rule_type: string;
  condition: {
    field?: string;
    condition?: string;
    value: string;
  }[];
  action: {
    action: string;
    message: string;
  };
  created_at: string;
  updated_at: string;
};
