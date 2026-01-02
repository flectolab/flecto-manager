import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { ApolloProvider } from '@apollo/client/react'
import { apolloClient } from './graphql/client'
import { ThemeProvider } from './contexts/ThemeContext'
import { AuthProvider } from './contexts/AuthContext'
import { ProtectedRoute } from './components/ProtectedRoute'
import { Layout } from './components/layout'
import { AdminLayout } from './components/admin'
import { NamespaceSelector } from './pages/NamespaceSelector'
import { Dashboard } from './pages/Dashboard'
import { Pages } from './pages/Pages'
import { PageForm } from './pages/PageForm'
import { Redirects } from './pages/Redirects'
import { RedirectForm } from './pages/RedirectForm'
import { RedirectTesterPage } from './pages/RedirectTesterPage'
import { Agents } from './pages/Agents'
import { Login } from './pages/Login'
import { LoginCallback } from './pages/LoginCallback'
import { Profile } from './pages/Profile'
import { AdminDashboard } from './pages/admin/Dashboard'
import { Users as AdminUsers } from './pages/admin/Users'
import { UserForm as AdminUserForm } from './pages/admin/UserForm'
import { Roles as AdminRoles } from './pages/admin/Roles'
import { RoleForm as AdminRoleForm } from './pages/admin/RoleForm'
import { Namespaces as AdminNamespaces } from './pages/admin/Namespaces'
import { NamespaceForm as AdminNamespaceForm } from './pages/admin/NamespaceForm'
import { Projects as AdminProjects } from './pages/admin/Projects'
import { ProjectForm as AdminProjectForm } from './pages/admin/ProjectForm'
import { Tokens as AdminTokens } from './pages/admin/Tokens'
import { TokenForm as AdminTokenForm } from './pages/admin/TokenForm'
import { NotFound } from './pages/NotFound'

function ProjectRoutes() {
  return (
    <Layout>
      <Routes>
        <Route index element={<Dashboard />} />
        <Route path="redirects" element={<Redirects />} />
        <Route path="redirects/add" element={<RedirectForm />} />
        <Route path="redirects/edit/:id" element={<RedirectForm />} />
        <Route path="redirect-tester" element={<RedirectTesterPage />} />
        <Route path="pages" element={<Pages />} />
        <Route path="pages/add" element={<PageForm />} />
        <Route path="pages/edit/:id" element={<PageForm />} />
        <Route path="agents" element={<Agents />} />
      </Routes>
    </Layout>
  )
}

function AdminRoutes() {
  return (
    <AdminLayout>
      <Routes>
        <Route index element={<AdminDashboard />} />
        <Route path="users" element={<AdminUsers />} />
        <Route path="users/new" element={<AdminUserForm />} />
        <Route path="users/:id" element={<AdminUserForm />} />
        <Route path="roles" element={<AdminRoles />} />
        <Route path="roles/new" element={<AdminRoleForm />} />
        <Route path="roles/:id" element={<AdminRoleForm />} />
        <Route path="namespaces" element={<AdminNamespaces />} />
        <Route path="namespaces/new" element={<AdminNamespaceForm />} />
        <Route path="namespaces/:id" element={<AdminNamespaceForm />} />
        <Route path="projects" element={<AdminProjects />} />
        <Route path="projects/new" element={<AdminProjectForm />} />
        <Route path="projects/:namespaceCode/:projectCode" element={<AdminProjectForm />} />
        <Route path="tokens" element={<AdminTokens />} />
        <Route path="tokens/new" element={<AdminTokenForm />} />
        <Route path="tokens/:id" element={<AdminTokenForm />} />
      </Routes>
    </AdminLayout>
  )
}

function App() {
  return (
    <ThemeProvider>
      <ApolloProvider client={apolloClient}>
        <AuthProvider>
          <BrowserRouter>
            <Routes>
              <Route path="/login" element={<Login />} />
              <Route path="/login/callback" element={<LoginCallback />} />
              <Route
                path="/"
                element={
                  <ProtectedRoute>
                    <NamespaceSelector />
                  </ProtectedRoute>
                }
              />
              <Route
                path="/profile"
                element={
                  <ProtectedRoute>
                    <Profile />
                  </ProtectedRoute>
                }
              />
              <Route
                path="/admin/*"
                element={
                  <ProtectedRoute requireAdmin>
                    <AdminRoutes />
                  </ProtectedRoute>
                }
              />
              <Route
                path="/:namespace/:project/*"
                element={
                  <ProtectedRoute>
                    <ProjectRoutes />
                  </ProtectedRoute>
                }
              />
              <Route
                path="*"
                element={
                  <ProtectedRoute>
                    <NotFound />
                  </ProtectedRoute>
                }
              />
            </Routes>
          </BrowserRouter>
        </AuthProvider>
      </ApolloProvider>
    </ThemeProvider>
  )
}

export default App
