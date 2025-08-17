/* eslint-disable @typescript-eslint/ban-ts-comment */
import React from "react";

class ErrorBoundary extends React.Component {
  constructor(props: any) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError() {
    return { hasError: true };
  }

  componentDidCatch(error: any, errorInfo: any) {
    console.error("Error:", error, errorInfo);
  }

  render() {
    //@ts-ignore
    if (this.state?.hasError) {
      return (
        <div className="min-h-screen flex items-center justify-center bg-gradient-to-b from-white to-primary/20 p-6">
          <div className="w-full max-w-md">
            <div className="bg-white rounded-2xl shadow-xl p-8 -mt-16">
              <div className="flex justify-center mb-8">
                <img src="/images/techx-logo.png" alt="TechX Logo" className="object-cover" />
              </div>
              <h2 className="text-2xl font-semibold text-center mb-8 text-primary inline-block w-full">
                Hệ thống đang có vấn đề
              </h2>

              {/* Reload page */}
              <button
                onClick={() => window.location.reload()}
                className="w-full py-3 px-4 bg-primary text-white rounded-xl hover:opacity-90 transition-opacity font-medium focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary"
              >
                Tải lại trang
              </button>
            </div>
          </div>
        </div>
      );
    }

    //@ts-ignore
    return this.props.children;
  }
}

export default ErrorBoundary;
