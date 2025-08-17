import { Result, Tag } from "antd";
import { CircleCheckBigIcon, CircleXIcon, ShieldCheckIcon, TriangleAlertIcon } from "lucide-react";

const ComplianceValidation = ({ rules_results }: { rules_results: any }) => {
  console.log("ComplianceValidation", rules_results);
  if (!rules_results)
    return (
      <div className="border-b border-gray-200 p-3">
        <div className="flex items-center gap-2 text-lg font-bold">
          <ShieldCheckIcon />
          <span>Compliance Overview</span>
        </div>
        <Result title="No rules found" subTitle="This template has no rules to check compliance" />
      </div>
    );
  const passedRules = rules_results.passed_rules?.length || 0;
  const warningRules = rules_results.warning_rules?.length || 0;
  const failedRules = rules_results.failed_rules?.length || 0;

  const totalRules = passedRules + warningRules + failedRules;

  const formatcheckedDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleDateString("en-US");
  };
  return (
    <>
      <div className="border-b border-gray-200 p-3">
        <div className="flex items-center gap-2 text-lg font-bold">
          <ShieldCheckIcon />
          <span>Compliance Overview</span>
        </div>
        <div className="flex items-center justify-evenly">
          <div className="text-center">
            <p className="text-2xl font-bold text-blue-500">{totalRules}</p>
            <p>Total Rules</p>
          </div>
          <div className="text-center">
            <p className="text-2xl font-bold text-green-500">{passedRules}</p>
            <p>Passed</p>
          </div>
          <div className="text-center">
            <p className="text-2xl font-bold text-yellow-500">{warningRules}</p>
            <p>Warnings</p>
          </div>
          <div className="text-center">
            <p className="text-2xl font-bold text-red-500">{failedRules}</p>
            <p>Failed</p>
          </div>
        </div>
        <div className="flex items-center justify-center mt-2 p-2 rounded-lg bg-red-500">
          <p className="text-lg font-bold text-white">Status: {rules_results.overall_status.toUpperCase()}</p>
        </div>
      </div>
      <div className="flex-1 p-3 pt-0">
        <p className="text-lg font-bold">Rule Validation Results</p>
        <div className="flex flex-col gap-2 overflow-y-auto">
          {passedRules > 0 &&
            rules_results.passed_rules.map((rule: any) => (
              <div className="border border-gray-200 rounded p-2" key={rule.rule_name}>
                <div className="flex items-center justify-between">
                  <span className="flex items-center gap-2">
                    <CircleCheckBigIcon color="green" />
                    {rule.rule_name}
                  </span>
                  <Tag color="green">Passed</Tag>
                </div>
                <p className="text-sm text-gray-500">{rule.message}</p>
                <p className="text-sm text-gray-500">
                  Affected Fields: <Tag color="blue">{rule.rule_type}</Tag>
                </p>
                <p className="text-sm text-gray-500">Checked: {formatcheckedDate(rule.timestamp)}</p>
              </div>
            ))}
          {warningRules > 0 &&
            rules_results.warning_rules.map((rule: any) => (
              <div className="border border-gray-200 rounded p-2" key={rule.rule_name}>
                <div className="flex items-center justify-between">
                  <span className="flex items-center gap-2">
                    <TriangleAlertIcon color="#d4b106" />
                    {rule.rule_name}
                  </span>
                  <Tag color="yellow">Warning</Tag>
                </div>
                <p className="text-sm text-gray-500">{rule.message}</p>
                <p className="text-sm text-gray-500">
                  Affected Fields: <Tag color="blue">{rule.rule_type}</Tag>
                </p>
                <p className="text-sm text-gray-500">Checked: {formatcheckedDate(rule.timestamp)}</p>
              </div>
            ))}
          {failedRules > 0 &&
            rules_results.failed_rules.map((rule: any) => (
              <div className="border border-gray-200 rounded p-2" key={rule.rule_name}>
                <div className="flex items-center justify-between">
                  <span className="flex items-center gap-2">
                    <CircleXIcon color="red" />
                    {rule.rule_name}
                  </span>
                  <Tag color="red">Failed</Tag>
                </div>
                <p className="text-sm text-gray-500">{rule.message}</p>
                <p className="text-sm text-gray-500">
                  Affected Fields: <Tag color="blue">{rule.rule_type}</Tag>
                </p>
                <p className="text-sm text-gray-500">Checked: {formatcheckedDate(rule.timestamp)}</p>
              </div>
            ))}
        </div>
      </div>
    </>
  );
};

export default ComplianceValidation;
