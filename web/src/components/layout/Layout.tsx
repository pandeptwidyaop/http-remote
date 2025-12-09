import { ReactNode } from 'react';
import { Github } from 'lucide-react';
import Sidebar from './Sidebar';

interface LayoutProps {
  children: ReactNode;
}

export default function Layout({ children }: LayoutProps) {
  return (
    <div className="min-h-screen bg-gray-100">
      <Sidebar />

      {/* Main Content - offset by sidebar width */}
      <div className="md:ml-64 transition-all duration-300 ease-in-out">
        {/* Mobile spacer for fixed header */}
        <div className="h-14 md:hidden" />

        <main className="min-h-[calc(100vh-56px)] md:min-h-screen p-4 md:p-6 lg:p-8">
          <div className="max-w-7xl mx-auto">{children}</div>
        </main>

        <footer className="bg-white border-t border-gray-200 py-4">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div className="flex flex-col sm:flex-row items-center justify-between gap-2 text-sm text-gray-500">
              <span>HTTP Remote - DevOps deployment tool</span>
              <a
                href="https://github.com/pandeptwidyaop/http-remote"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-2 text-gray-500 hover:text-gray-900 transition-colors"
              >
                <Github className="w-4 h-4" />
                <span>View on GitHub</span>
              </a>
            </div>
          </div>
        </footer>
      </div>
    </div>
  );
}
