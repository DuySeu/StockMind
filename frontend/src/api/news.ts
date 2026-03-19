import api from ".";

export const getLatestNews = async () => {
  const response = await api.get(`/news`);
  return response.data;
};
