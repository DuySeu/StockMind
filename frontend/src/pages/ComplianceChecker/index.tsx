import { Button, DatePicker, Form, Input, Modal, Popconfirm, Select, Table, TableProps, Typography } from "antd";
import { useEffect, useState } from "react";
import { createRule, deleteRule, getRules, updateRule } from "../../api/rules-api";
import { useNotificationService } from "../../hooks/Notification";
import PlusIcon from "../../assets/icons/PlusIcon";
import { Rule } from "../../types/rule";
import { ruleActions, ruleConditions, ruleFields, ruleTypes } from "./rule_data";
import dayjs from "dayjs";
import { TrashIcon } from "lucide-react";

function ComplianceChecker() {
  const { createErrorNotification, createSuccessNotification } = useNotificationService();

  const [isModalOpen, setIsModalOpen] = useState(false);
  const [form] = Form.useForm();
  const [rules, setRules] = useState<Rule[]>([]);
  const [loading, setLoading] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [editingRuleKey, setEditingRuleKey] = useState<string | null>(null);

  // Fetch rules on component mount
  useEffect(() => {
    fetchRules();
  }, [createErrorNotification]);

  console.log(rules);

  // Update columns to match the new data structure
  const columns: TableProps<Rule>["columns"] = [
    {
      title: "Rule Name",
      dataIndex: "name",
      key: "name",
      render: (text) => <a>{text}</a>,
    },
    {
      title: "Description",
      dataIndex: "description",
      key: "description",
      render: (text) => text || "No description available",
    },
    {
      title: "Type",
      dataIndex: "rule_type",
      key: "rule_type",
      render: (text) => <p>{text}</p>,
    },
    {
      title: "Action",
      key: "action",
      render: (_, record) => (
        <div onClick={(e) => e.stopPropagation()}>
          <Popconfirm
            title="Delete the rule"
            description="Are you sure to delete this rule?"
            onConfirm={() => handleDelete(record.id)}
            okText="Yes"
            cancelText="No"
          >
            <Button danger icon={<TrashIcon size={16} />} type="text">
              Delete
            </Button>
          </Popconfirm>
        </div>
      ),
    },
  ];

  const fetchRules = async () => {
    try {
      setLoading(true);
      const response = await getRules();

      setRules(response);
    } catch (error) {
      console.error("Error fetching rules:", error);
      createErrorNotification("Failed to fetch rules");
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (key: string) => {
    try {
      await deleteRule(key);
      createSuccessNotification("Rule deleted successfully");
      fetchRules(); // Refresh the list
    } catch (error) {
      console.error("Error deleting rule:", error);
      createErrorNotification("Failed to delete rule");
    }
  };

  const handleEdit = (record: Rule) => {
    setIsModalOpen(true);
    setIsEditing(true);
    setEditingRuleKey(record.id);
    // Convert date string to dayjs object if it's a date field
    const ruleField = record.condition[0]?.field || ruleFields[0];
    const ruleValue = record.condition[0]?.value || "";
    let convertedValue = ruleValue || null;

    if (ruleField === "Expiry Date" || ruleField === "Issued Date") {
      convertedValue = (dayjs(ruleValue) as unknown as string) || null;
    } else {
      convertedValue = ruleValue?.toString() || null;
    }

    form.setFieldsValue({
      ruleName: record.name,
      ruleDescription: record.description,
      ruleType: record.rule_type,
      ruleField: ruleField,
      ruleCondition: record.condition[0]?.condition || ruleConditions[0],
      ruleValue: convertedValue,
      ruleAction: record.action.action.toLowerCase(),
      ruleMessage: record.action.message,
    });
  };

  const showModal = () => {
    setIsModalOpen(true);
    setIsEditing(false);
    setEditingRuleKey(null);
    form.resetFields();
  };

  const handleCancel = () => {
    setIsModalOpen(false);
    form.resetFields();
  };

  const onFinish = async (values: any) => {
    try {
      const payload = {
        name: values.ruleName,
        description: values.ruleDescription || "",
        rule_type: values.ruleType,
        condition: [
          {
            field: values.ruleField,
            condition: values.ruleCondition,
            value: values.ruleValue,
          },
        ],
        action: { action: values.ruleAction, message: values.ruleMessage },
      };

      if (isEditing && editingRuleKey) {
        await updateRule(editingRuleKey, payload);
        createSuccessNotification("Rule updated successfully");
      } else {
        await createRule(payload);
        createSuccessNotification("Rule created successfully");
      }

      setIsModalOpen(false);
      form.resetFields();
      fetchRules();
    } catch {
      createErrorNotification(isEditing ? "Failed to update rule" : "Failed to create rule");
    }
  };

  const renderRuleValue = (ruleType: string) => {
    switch (ruleType) {
      case "Mandatory Fields":
        return (
          <Form.Item name="ruleValue">
            <Input placeholder="e.g., expiryDate" />
          </Form.Item>
        );
      default:
        return (
          <div className="flex gap-2">
            <Form.Item name="ruleField" className="w-1/3">
              <Select
                options={ruleFields.map((field) => ({ label: field, value: field }))}
                onChange={(value) => {
                  // Reset ruleCondition when ruleField changes
                  const availableConditions = renderRuleCondition(value);
                  if (availableConditions.length > 0) {
                    form.setFieldValue("ruleCondition", availableConditions[0]);
                  }
                }}
              />
            </Form.Item>
            <Form.Item name="ruleCondition" className="w-1/3" dependencies={["ruleField"]}>
              <Select
                options={renderRuleCondition(form.getFieldValue("ruleField") as string).map((condition) => ({
                  label: condition,
                  value: condition,
                }))}
              />
            </Form.Item>
            <Form.Item name="ruleValue" className="w-1/3" dependencies={["ruleField", "ruleCondition"]}>
              {renderRuleValueField(
                form.getFieldValue("ruleField") as string,
                form.getFieldValue("ruleCondition") as string
              )}
            </Form.Item>
          </div>
        );
    }
  };

  const renderRuleCondition = (ruleField: string) => {
    switch (ruleField) {
      case "Full Name":
      case "LC Number":
      case "Port Name":
        return ruleConditions.filter(
          (condition) =>
            condition === "Equal" ||
            condition === "Not Equal" ||
            condition === "Contains" ||
            condition === "Not Contain"
        );
      case "Expiry Date":
      case "Issued Date":
        return ruleConditions.filter(
          (condition) =>
            condition === "Greater Than" ||
            condition === "Less Than" ||
            condition === "Greater Than or Equal" ||
            condition === "Less Than or Equal" ||
            condition === "In Range"
        );
      default:
        return ruleConditions;
    }
  };

  const renderRuleValueField = (ruleField: string, ruleCondition: string) => {
    if (ruleField === "Expiry Date" || ruleField === "Issued Date") {
      if (ruleCondition === "In Range") {
        return <DatePicker.RangePicker />;
      }
      return <DatePicker className="w-full" />;
    }
    return <Input placeholder="Input value" />;
  };

  return (
    <>
      <div className="border border-gray-200 rounded-lg p-6">
        <div className="flex justify-between gap-2 m-2">
          <Typography.Title className="!text-primary !text-[24px] !font-bold !mb-0">
            Compliance Rules List
          </Typography.Title>
          <Button type="primary" icon={<PlusIcon />} onClick={showModal}>
            Create Rule
          </Button>
        </div>

        <Table<Rule>
          columns={columns}
          dataSource={rules}
          loading={loading}
          rowKey="id"
          onRow={(record) => ({
            onClick: () => {
              handleEdit(record);
              setIsModalOpen(true);
            },
            style: { cursor: "pointer" },
          })}
          pagination={
            rules.length > 10 && {
              pageSize: 10,
              showSizeChanger: false,
              showQuickJumper: false,
              showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} rules`,
            }
          }
        />
      </div>

      {/* Create Rule Modal */}
      <Modal
        title={isEditing ? "Edit Rule" : "Create New Rule"}
        closable={{ "aria-label": "Custom Close Button" }}
        open={isModalOpen}
        onCancel={handleCancel}
        footer={null}
        width={800}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={onFinish}
          initialValues={{
            ruleType: ruleTypes[0],
            ruleAction: ruleActions[0],
            ruleField: ruleFields[0],
            ruleCondition: renderRuleCondition(ruleFields[0])[0],
          }}
        >
          <Form.Item label="Rule Name" name="ruleName" rules={[{ required: true, message: "Please input rule name!" }]}>
            <Input />
          </Form.Item>
          <Form.Item label="Rule Description" name="ruleDescription">
            <Input.TextArea autoSize={{ minRows: 1, maxRows: 3 }} />
          </Form.Item>
          <Form.Item
            label="Rule Type"
            name="ruleType"
            rules={[{ required: true, message: "Please select rule type!" }]}
          >
            <Select options={ruleTypes.map((type) => ({ label: type, value: type }))} />
          </Form.Item>
          <Form.Item
            noStyle
            shouldUpdate={(prevValues, currentValues) =>
              prevValues.ruleType !== currentValues.ruleType ||
              prevValues.ruleField !== currentValues.ruleField ||
              prevValues.ruleCondition !== currentValues.ruleCondition
            }
          >
            {({ getFieldValue }) => (
              <div className="flex flex-col gap-2 mb-2">
                <p className="text-sm">Rule Condition</p>
                {renderRuleValue(getFieldValue("ruleType") as string)}
              </div>
            )}
          </Form.Item>
          <div className="flex flex-col gap-2">
            <p className="text-sm">Action</p>
            <div className="flex gap-2">
              <Form.Item name="ruleAction" className="w-1/4">
                <Select options={ruleActions.map((action) => ({ label: action, value: action }))} />
              </Form.Item>
              <Form.Item name="ruleMessage" className="w-3/4">
                <Input placeholder="Action message(This message will be shown when the rule is triggered)" />
              </Form.Item>
            </div>
          </div>
          <div className="flex justify-end gap-2">
            <Button type="primary" htmlType="submit" style={{ marginTop: 8 }}>
              {isEditing ? "Update Rule" : "Create Rule"}
            </Button>
          </div>
        </Form>
      </Modal>
    </>
  );
}

export default ComplianceChecker;
