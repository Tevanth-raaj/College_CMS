import React from "react";

const MainLayout = ({ children, title, subtitle, actions }) => {
  return (
    <div className="flex-1">
      <header className="bg-white border-b border-gray-200 sticky top-0 z-40">
        <div className="px-4 h-[79px] flex items-center justify-between">
          <div className="flex flex-col">
            {title && <h1 className="text-xl font-bold text-gray-900">{title}</h1>}
            {subtitle && <p className="text-sm text-gray-500 mt-0.5">{subtitle}</p>}
          </div>
          {actions && <div className="flex items-center space-x-3">{actions}</div>}
        </div>
      </header>

      <main className="p-4">{children}</main>
    </div>
  );
};

export default MainLayout;
