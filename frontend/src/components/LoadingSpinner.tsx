import { Spin } from "antd";

function LoadingSpinner() {
  return (
    <div className="w-full h-full flex items-center justify-center">
      <Spin />
    </div>
  );
}

export default LoadingSpinner;
