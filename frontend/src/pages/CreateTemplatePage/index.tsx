import { Button, Form, Input, Modal, Popconfirm, Select, Table, TableProps, Typography } from "antd";
import { useEffect, useState } from "react";
import { getRules } from "../../api/rules-api";
import { createTemplate, deleteTemplate, getTemplates, updateTemplate } from "../../api/template-api";
import PlusIcon from "../../assets/icons/PlusIcon";
import { JsonEditor } from "../../components";
import { useNotificationService } from "../../hooks/Notification";
import { Rule } from "../../types/rule";
import { Template } from "../../types/template";
import { TrashIcon } from "lucide-react";

function Templates() {
  const { createErrorNotification, createSuccessNotification } = useNotificationService();

  const [isModalOpen, setIsModalOpen] = useState(false);
  const [form] = Form.useForm();
  const [templates, setTemplates] = useState<Template[]>([]);
  const [rules, setRules] = useState<Rule[]>([]);
  const [loading, setLoading] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [editingTemplateKey, setEditingTemplateKey] = useState<string | null>(null);
  // Fetch templates on component mount
  useEffect(() => {
    const fetchRules = async () => {
      try {
        const response = await getRules();
        setRules(response);
      } catch (error) {
        console.error("Error fetching rules:", error);
        createErrorNotification("Failed to fetch rules");
      }
    };
    fetchRules();
    fetchTemplates();
  }, [createErrorNotification]);

  // Update columns to match the new data structure
  const columns: TableProps<Template>["columns"] = [
    {
      title: "Template Name",
      dataIndex: "name",
      key: "name",
      render: (text) => <a>{text}</a>,
    },
    {
      title: "Description",
      dataIndex: "description",
      key: "description",
      render: (text) => text || "-",
    },
    {
      title: "Rules",
      key: "rule_ids",
      dataIndex: "rule_ids",
      render: (text) => (
        <div style={{ maxWidth: 300, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
          {text.map((rule: string) => rules.find((r) => r.id === rule)?.name).join(", ")}
        </div>
      ),
    },
    {
      title: "Created At",
      dataIndex: "created_at",
      key: "created_at",
      render: (text) => <span>{new Date(text).toLocaleString()}</span>,
    },
    {
      title: "Action",
      key: "action",
      render: (_, record) => (
        <div onClick={(e) => e.stopPropagation()}>
          <Popconfirm
            title="Delete the template"
            description="Are you sure to delete this template?"
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

  const fetchTemplates = async () => {
    try {
      setLoading(true);
      const response = await getTemplates();
      setTemplates(response);
    } catch (error) {
      console.error("Error fetching templates:", error);
      createErrorNotification("Failed to fetch templates");
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (key: string) => {
    try {
      await deleteTemplate(key);
      createSuccessNotification("Template deleted successfully");
      fetchTemplates();
    } catch (error) {
      console.error("Error deleting template:", error);
      createErrorNotification("Failed to delete template");
    }
  };

  const handleEdit = (record: Template) => {
    setIsModalOpen(true);
    setIsEditing(true);
    setEditingTemplateKey(record.id);
    form.setFieldsValue({
      templateName: record.name,
      templateDescription: record.description,
      templatePrompt: record.prompt,
      templateField: JSON.stringify(record.field, null, 2),
      templateRules: record.rule_ids,
    });
  };

  const showModal = () => {
    setIsModalOpen(true);
    setIsEditing(false);
    setEditingTemplateKey(null);
    form.resetFields();
  };

  const handleCancel = () => {
    setIsModalOpen(false);
    form.resetFields();
  };

  const onFinish = async (values: any) => {
    try {
      const payload = {
        name: values.templateName,
        description: values.templateDescription,
        prompt: values.templatePrompt,
        field: JSON.parse(values.templateField),
        rule_ids: values.templateRules || [],
      };
      if (isEditing && editingTemplateKey) {
        await updateTemplate(editingTemplateKey, payload);
        createSuccessNotification("Template updated successfully");
      } else {
        await createTemplate(payload);
        createSuccessNotification("Template created successfully");
      }

      setIsModalOpen(false);
      form.resetFields();
      fetchTemplates();
    } catch {
      createErrorNotification(isEditing ? "Failed to update template" : "Failed to create template");
    }
  };

  return (
    <>
      <div className="border border-gray-200 rounded-lg p-6">
        <div className="flex justify-between gap-2 m-4">
          <Typography.Title className="!text-primary !text-[24px] !font-bold !mb-0">Templates List</Typography.Title>
          <Button type="primary" icon={<PlusIcon />} onClick={showModal}>
            Create New Template
          </Button>
        </div>

        <Table<Template>
          columns={columns}
          dataSource={templates}
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
            templates.length > 10 && {
              pageSize: 10,
              showSizeChanger: false,
              showQuickJumper: false,
              showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} templates`,
            }
          }
        />
      </div>

      {/* Create Template Modal */}
      <Modal
        title={isEditing ? "Edit Template" : "Create New Template"}
        closable={{ "aria-label": "Custom Close Button" }}
        open={isModalOpen}
        onCancel={handleCancel}
        footer={null}
      >
        <Form form={form} layout="vertical" onFinish={onFinish}>
          <Form.Item
            label="Template Name"
            name="templateName"
            rules={[{ required: true, message: "Please input template name!" }]}
          >
            <Input />
          </Form.Item>
          <Form.Item label="Template Description" name="templateDescription">
            <Input.TextArea autoSize={{ minRows: 1, maxRows: 3 }} />
          </Form.Item>
          <Form.Item label="Template Rules" name="templateRules">
            <Select
              mode="multiple"
              allowClear
              placeholder="Select rules"
              options={rules.map((rule) => ({ label: rule.name, value: rule.id }))}
              showSearch
              filterOption={(input, option) => (option?.label ?? "").toLowerCase().includes(input.toLowerCase())}
              optionFilterProp="label"
            />
          </Form.Item>
          <Form.Item
            label="Template Prompt"
            name="templatePrompt"
            rules={[{ required: true, message: "Please input template prompt!" }]}
          >
            <Input.TextArea autoSize={{ minRows: 3, maxRows: 3 }} />
          </Form.Item>
          <Form.Item name="templateField" label="Template Fields (JSON)">
            <JsonEditor
              placeholder={`{
  "field1": "value1",
  "field2": "value2"
}`}
            />
          </Form.Item>
          <Button type="primary" htmlType="submit">
            {isEditing ? "Update Template" : "Create Template"}
          </Button>
        </Form>
      </Modal>
    </>
  );
}

export default Templates;
