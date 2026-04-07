import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import { Search } from 'lucide-svelte';
import Icon from '../Icon.svelte';
describe('Icon', () => {
    it('renders without throwing', () => {
        const { container } = render(Icon, { props: { icon: Search } });
        expect(container).toBeTruthy();
    });
    it('accepts size prop', () => {
        const { container } = render(Icon, { props: { icon: Search, size: 24 } });
        expect(container).toBeTruthy();
    });
});
