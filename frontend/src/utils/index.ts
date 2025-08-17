export const convertNameForConversation = (name: string) => {
  return name.length <= 30 ? name.trim() : name.slice(0, 20).trim() + "...";
};

export const formatLabel = (key: string) => {
  return key
    .replace(/([A-Z])/g, " $1") // Add space before uppercase letters (camelCase)
    .replace(/_/g, " ") // Replace underscores with spaces (snake_case)
    .trim()
    .replace(/\b\w/g, (char) => char.toUpperCase()); // Capitalize first letter of each word
};

export const convertToNestedObject = (
  flatObject: Record<string, any>
): Record<string, any> => {
  const nestedObject: Record<string, any> = {};

  Object.entries(flatObject).forEach(([key, value]) => {
    const keys = key.split(".");
    let current = nestedObject;

    keys.forEach((k, index) => {
      const arrayMatch = k.match(/^(\w+)\[(\d+)]$/);
      if (arrayMatch) {
        const arrayKey = arrayMatch[1];
        const arrayIndex = parseInt(arrayMatch[2], 10);

        if (!current[arrayKey]) {
          current[arrayKey] = [];
        }

        current = current[arrayKey];
        current[arrayIndex] = value;
      } else {
        if (!current[k]) {
          current[k] = index === keys.length - 1 ? value : {};
        }
        current = current[k];
      }
    });
  });

  return nestedObject;
};

export const flattenData = (data: any, parentKey = ""): Record<string, any> => {
  const flattened: Record<string, any> = {};

  Object.entries(data).forEach(([key, value]) => {
    const fieldName = parentKey ? `${parentKey}.${key}` : key;

    if (typeof value === "object" && value !== null && !Array.isArray(value)) {
      Object.assign(flattened, flattenData(value, fieldName));
    } else {
      flattened[fieldName] = Array.isArray(value)
        ? value.join(", ")
        : value || "";
    }
  });

  return flattened;
};
