{%allowmissingval%}{% if nothing %}false{% else %}true{% endif %}{%endallowmissingval%}
{% if simple %}simple != nil{% endif %}
{% if simple.uint %}uint != 0{% endif %}
{% if simple.float %}float != 0.0{% endif %}
{% if !simple %}false{% else %}!simple{% endif %}
{% if !simple.uint %}false{% else %}!simple.uint{% endif %}
{% if !simple.float %}false{% else %}!simple.float{% endif %}
{% if "Text" in complex.post %}text field in complex.post{% endif %}
{% if 5 in simple.intmap %}5 in simple.intmap{% endif %}
{% if simple.uint in simple.multiple_item_list %}simple.uint in simple.multiple_item_list{% endif %}
{% if !0.0 %}!0.0{% endif %}
{% if !0 %}!0{% endif %}
{% if not complex.post %}true{% else %}false{% endif %}
{% if simple.number == 43 %}no{% else %}42{% endif %}
{% if simple.number < 42 %}false{% elif simple.number > 42 %}no{% elif simple.number >= 42 %}yes{% else %}no{% endif %}
{% if simple.number < 42 %}false{% elif simple.number > 42 %}no{% elif simple.number != 42 %}no{% else %}yes{% endif %}
{%allowmissingval%}{% if 0 %}!0{% elif nothing %}nothing{% else %}true{% endif %}{%endallowmissingval%}
{% if 0 %}!0{% elif simple.float %}simple.float{% else %}false{% endif %}
{% if 0 %}!0{% elif !simple.float %}false{% elif "Text" in complex.post%}Elseif with no else{% endif %}