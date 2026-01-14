import type { RouteRecordRaw } from 'vue-router';
import HomeView from '@/views/HomeView.vue';
import Depot from '@/views/Depot.vue';
import AdminManagement from '@/views/AdminManagement.vue';

// Define the routes for the generic routes
export const basicRoutes: RouteRecordRaw[] = [
    {
        path: '/',
        name: 'home',
        component: HomeView,
        children: [
            {
                path: '/',
                name: 'home-index',
                component: Depot
            },
            {
                path: '/admin',
                name: 'admin-management',
                component: AdminManagement
            },
        ]
    }
];
