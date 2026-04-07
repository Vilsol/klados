import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import DropdownMenuTest from './DropdownMenuTest.svelte';
describe('DropdownMenu', () => {
    it('renders trigger without throwing', () => {
        const { getByText } = render(DropdownMenuTest);
        expect(getByText('Open menu')).toBeTruthy();
    });
});
