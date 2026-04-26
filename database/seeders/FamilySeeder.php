<?php

namespace Database\Seeders;

use App\Models\Person;
use Illuminate\Database\Seeder;

class FamilySeeder extends Seeder
{
    /**
     * Seed the four family members in display order.
     */
    public function run(): void
    {
        $members = [
            ['name' => '爸爸', 'display_order' => 1],
            ['name' => '妈妈', 'display_order' => 2],
            ['name' => '可遇', 'display_order' => 3],
            ['name' => '奇乐', 'display_order' => 4],
        ];

        foreach ($members as $member) {
            Person::firstOrCreate(
                ['name' => $member['name']],
                ['display_order' => $member['display_order']],
            );
        }
    }
}
