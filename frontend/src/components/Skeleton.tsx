import { Skeleton, SkeletonProps } from "antd";

function SkeletonLoading(props: SkeletonProps) {
  return <Skeleton active {...props} />;
}

export default SkeletonLoading;
