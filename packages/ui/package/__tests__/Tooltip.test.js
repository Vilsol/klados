import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import TooltipTest from './TooltipTest.svelte';
describe('Tooltip', () => {
    it('renders trigger without throwing', () => {
        const { container } = render(TooltipTest);
        expect(container).toBeTruthy();
    });
});
